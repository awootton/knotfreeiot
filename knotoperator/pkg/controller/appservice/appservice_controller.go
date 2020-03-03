package appservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/knotoperator/pkg/apis/app/v1alpha1"
	appv1alpha1 "github.com/awootton/knotfreeiot/knotoperator/pkg/apis/app/v1alpha1"
	"github.com/awootton/knotfreeiot/kubectl"
	"github.com/go-logr/logr"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/awootton/knotfreeiot/iot"
)

var log = logf.Log.WithName("controller_appservice_knotfree")

/**
* atw ha ha lUSER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new AppService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAppService{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("appservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AppService
	err = c.Watch(&source.Kind{Type: &appv1alpha1.AppService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TO DO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner AppService
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1alpha1.AppService{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileAppService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAppService{}

var count int

// ReconcileAppService reconciles a AppService object
type ReconcileAppService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// LastClusterState is sort of a cache
var LastClusterState *appv1alpha1.ClusterState

var reconcileLogger logr.Logger = log

// PrintMe is needing work
func PrintMe(msg string, args ...interface{}) {
	if reconcileLogger != nil {
		if len(args)&1 == 0 {
			reconcileLogger.Info(msg, "dummy", args)
		} else {
			reconcileLogger.Info(msg, args)
		}
	}
}

// Reconcile reads that state of the cluster for a AppService object and makes changes based on the state read
// and what is in the AppService.Spec
// Note: fixme too long
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAppService) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reconcileLogger = reqLogger
	reqLogger.Info("Reconciling AppService")

	PrintMe("count", "n", count) // atw fixme: this is a stat, not a log
	count++

	_appchanged := false
	appchanged := func() {
		_appchanged = true
	}
	triggerGuruRebalance := false

	// Fetch the AppService instance
	instance := &appv1alpha1.AppService{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			PrintMe("AppService NOT FOUND") // atw fixme: this is a error, not a log
			rr := reconcile.Result{}
			rr.RequeueAfter = 10 * time.Second
			return rr, nil
		}
		// Error reading the object - requeue the request.
		PrintMe("AppService FAIL", "err", err)
		return reconcile.Result{}, err
	}
	virginInstance := instance
	instance = instance.DeepCopy()

	if instance.Status.Ce == nil {
		instance.Status.Ce = v1alpha1.NewClusterState()
		//appchanged()
	} else if len(instance.Status.Ce.GuruNames) == 0 {
		instance.Status.Ce = v1alpha1.NewClusterState()
		appchanged()
	}
	if instance.Spec.Ce == nil {
		instance.Spec.Ce = v1alpha1.NewClusterState()
		appchanged()
	} else if len(instance.Spec.Ce.GuruNames) == 0 {
		instance.Spec.Ce = v1alpha1.NewClusterState()
		appchanged()
	}
	triggerGuruRebalance = _appchanged

	LastClusterState = instance.Spec.Ce

	//PrintMe("guru ", "names", instance.Spec.Ce.GuruNames)
	//namespace := request.Namespace

	items := &corev1.PodList{}
	err2 := r.client.List(context.TODO(), items)
	if err2 != nil {
		return reconcile.Result{}, err2
	}
	aidePods := make(map[string]corev1.Pod, 0)
	guruPods := make(map[string]corev1.Pod, 0)
	otherPods := make(map[string]corev1.Pod, 0)

	for i, pod := range items.Items {
		pname := pod.GetName()

		if strings.HasPrefix(pname, "aide-") {
			aidePods[pname] = pod
		} else if strings.HasPrefix(pname, "guru-") {
			guruPods[pname] = pod
		} else {
			otherPods[pname] = pod
		}
		_ = i
	}

	// are there nodes on the AppService list that don't exist as pods?
	for name, spec := range instance.Spec.Ce.Nodes {
		if strings.HasPrefix(name, "aide-") {
			_, ok := aidePods[name]
			if ok == false {
				delete(instance.Spec.Ce.Nodes, name)
				appchanged()
			}
		} else if strings.HasPrefix(name, "guru-") {
			_, ok := guruPods[name]
			if ok == false {
				delete(instance.Spec.Ce.Nodes, name)
				appchanged()
				triggerGuruRebalance = true
				for i, str := range instance.Spec.Ce.GuruNames {
					if str == name {
						// remove from GuruNames also which creates a big rebalance.
						lm1 := len(instance.Spec.Ce.GuruNames) - 1
						instance.Spec.Ce.GuruNames[i] = instance.Spec.Ce.GuruNames[lm1]
						instance.Spec.Ce.GuruNames = instance.Spec.Ce.GuruNames[0:lm1]
						break
					}
				}
			}
		}
		_ = spec
	}

	// clean up guru names on the AppService list
	for _, name := range instance.Spec.Ce.GuruNames {
		if strings.HasPrefix(name, "aide-") {
			PrintMe("HOW did an AIDE get on this list?")
		} else if strings.HasPrefix(name, "guru-") {
			_, ok := guruPods[name]
			if ok == false {
				delete(instance.Spec.Ce.Nodes, name)
				appchanged()
				triggerGuruRebalance = true
				for i, str := range instance.Spec.Ce.GuruNames {
					if str == name {
						// remove from GuruNames also which creates a big rebalance.
						lm1 := len(instance.Spec.Ce.GuruNames) - 1
						instance.Spec.Ce.GuruNames[i] = instance.Spec.Ce.GuruNames[lm1]
						instance.Spec.Ce.GuruNames = instance.Spec.Ce.GuruNames[0:lm1]
						break
					}
				}
			}
		}
	}

	// are there any non-pending guru's that are not on the AppService list?
	// what if it's on the list but it's not feeling well?
	for i, pod := range guruPods {
		nodeStats, present := instance.Spec.Ce.Nodes[pod.Name]
		if present == false {
			// so it's not on the list.
			// is it running?
			ready := false
			if len(pod.Status.ContainerStatuses) != 0 {
				ready = pod.Status.ContainerStatuses[0].Ready
			}
			if ready && len(pod.Status.PodIP) > 0 {

				stats := new(iot.ExecutiveStats)
				stats.TCPAddress = pod.Status.PodIP + ":8384"
				stats.HTTPAddress = pod.Status.PodIP + ":8080"
				stats.Name = pod.Name

				instance.Spec.Ce.Nodes[pod.Name] = stats

				instance.Spec.Ce.GuruNames = append(instance.Spec.Ce.GuruNames, pod.Name)

				appchanged()
				triggerGuruRebalance = true
			}
		} else if len(pod.Status.ContainerStatuses) != 0 {
			// we found it.
			// check the address
			if pod.Status.ContainerStatuses[0].Ready && len(pod.Status.PodIP) > 0 {
				nodeStats.TCPAddress = pod.Status.PodIP + ":8384"
				nodeStats.HTTPAddress = pod.Status.PodIP + ":8080"
			}
		}
		_ = i
	}

	// are lets also walk the aides list
	for i, pod := range aidePods {
		nodeStats, present := instance.Spec.Ce.Nodes[pod.Name]
		if present == false {
			// so it's not on the list.
			// is it running?
			ready := false
			if len(pod.Status.ContainerStatuses) != 0 {
				ready = pod.Status.ContainerStatuses[0].Ready
			}
			if ready && len(pod.Status.PodIP) > 0 {

				stats := new(iot.ExecutiveStats)
				stats.TCPAddress = pod.Status.PodIP + ":8384"
				stats.HTTPAddress = pod.Status.PodIP + ":8080"
				stats.Name = pod.Name
				instance.Spec.Ce.Nodes[pod.Name] = stats

				appchanged()
				triggerGuruRebalance = true // FIXME: We really only need to send the gurunames to this pod not everyone.
			}
		} else if len(pod.Status.ContainerStatuses) != 0 {
			// we found it.
			// check the address
			if pod.Status.ContainerStatuses[0].Ready && len(pod.Status.PodIP) > 0 {
				nodeStats.TCPAddress = pod.Status.PodIP + ":8384"
				nodeStats.HTTPAddress = pod.Status.PodIP + ":8080"
			}
		}
		_ = i
	}

	// http to all the nodes (aides and gurus) and get their status
	var wg = sync.WaitGroup{}
	var mux = sync.Mutex{}

	for i, pod := range aidePods {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func(name, add string) {
			defer wg.Done()
			es, err := GetServerStats(name, add)
			mux.Lock()
			defer mux.Unlock()
			if err != nil {
				delete(instance.Spec.Ce.Nodes, name)
			} else {
				nodeStats, present := instance.Spec.Ce.Nodes[name]
				if present {
					es.TCPAddress = nodeStats.TCPAddress
					es.HTTPAddress = nodeStats.HTTPAddress
					if !reflect.DeepEqual(nodeStats, es) {
						instance.Spec.Ce.Nodes[name] = es
						//appchanged()
					}
				}
			}
		}(pod.Name, add)
		_ = i
	}
	for i, pod := range guruPods {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func(name, add string) {
			defer wg.Done()
			es, err := GetServerStats(name, add)
			mux.Lock()
			defer mux.Unlock()
			if err != nil {
				delete(instance.Spec.Ce.Nodes, name)
			} else {
				nodeStats, present := instance.Spec.Ce.Nodes[name]
				es.TCPAddress = nodeStats.TCPAddress
				es.HTTPAddress = nodeStats.HTTPAddress
				if present {
					if !reflect.DeepEqual(nodeStats, es) {
						instance.Spec.Ce.Nodes[name] = es
						//appchanged()
					}
				}
			}
		}(pod.Name, add)
		_ = i
	}
	wg.Wait()

	// so now we have all the stats
	aideList := make([]*iot.ExecutiveStats, 0)
	guruList := make([]*iot.ExecutiveStats, 0)

	for key, val := range instance.Spec.Ce.Nodes {
		if strings.HasPrefix(val.Name, "aide-") {
			aideList = append(aideList, val)
		} else {
			guruList = append(guruList, val)
		}
		_ = key
	}

	resize := iot.CalcExpansionDesired(aideList, guruList)

	if resize.ChangeGurus != 0 {
		triggerGuruRebalance = true
	}

	if resize.ChangeGurus > 0 || len(guruList) == 0 { //} len(gurus) < neededGurus {
		// make a new one
		pod := newPodForCR(instance)
		// Set AppService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			PrintMe("SetControllerReference ", "err", err)
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			PrintMe("pod create fail", "err", err)
			return reconcile.Result{}, err
		}
		// don't modify the app yet.
		//instance.Spec.GuruNames = append(instance.Spec.GuruNames, pod.Name)
		//address := pod.Status.PodIP
		//PrintMe("new guru has ip ", address)
		//instance.Spec.GuruAddresses = append(instance.Spec.GuruAddresses, address+":8384")
		//appchanged()
	} else if resize.ChangeGurus < 0 {

		// delete the last one
		name := instance.Spec.Ce.GuruNames[len(instance.Spec.Ce.GuruNames)-1]
		pod, ok := guruPods[name]
		if ok {
			err := r.client.Delete(context.TODO(), &pod)
			if err != nil {
				PrintMe("pod delete fail", "err", err)
				return reconcile.Result{}, err
			}
		}
		mux.Lock()

		instance.Spec.Ce.GuruNames = instance.Spec.Ce.GuruNames[0 : len(instance.Spec.Ce.GuruNames)-1]
		delete(instance.Spec.Ce.Nodes, name)
		appchanged()
		defer mux.Unlock()
	}

	// items2 := &corev1.NodeList{} // forbidden on google gce
	// err3 := r.client.List(context.TODO(), items2)
	// if err3 == nil {
	// 	for i, node := range items2.Items {
	// 		nname := node.GetName()
	// 		PrintMe("node/", "name", nname)
	// 		_ = i
	// 	}
	// }

	// can  we get the replicas of the deployment?
	// List Deployments
	// Using a unstructured object.
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "Deployment",
		Version: "v1",
	})
	ckey := client.ObjectKey{
		Namespace: "knotspace",
		Name:      "aide",
	}
	geterr := r.client.Get(context.TODO(), ckey, u)
	if geterr != nil {
		PrintMe("depl list get err", "err", geterr)
		return reconcile.Result{}, err
	}
	name, ok, err := unstructured.NestedString(u.UnstructuredContent(), "metadata", "name")
	//PrintMe("get got name ", name, ok, err)

	repcount, ok, err := unstructured.NestedInt64(u.UnstructuredContent(), "spec", "replicas")
	//PrintMe("get got repcount ", repcount, ok, err)
	_ = name
	_ = ok

	targetRepCount := int64(len(aidePods))
	if resize.ChangeAides > 0 {
		targetRepCount++
	} else if resize.ChangeAides < 0 {
		targetRepCount--
	}
	if targetRepCount <= 0 {
		targetRepCount = 1
	}

	if geterr == nil && repcount != targetRepCount {
		unstructured.SetNestedField(u.UnstructuredContent(), targetRepCount, "spec", "replicas")
		err = r.client.Update(context.TODO(), u)
		if err != nil {
			PrintMe("update err", "err", err)
			return reconcile.Result{}, err
		}
	}

	if _appchanged {
		PrintMe("UPDATING app")

		podJSON, err := json.Marshal(virginInstance)
		if err != nil {
			PrintMe("sss", "err", err)
			return reconcile.Result{}, err
		}
		newPodJSON, err := json.Marshal(instance)
		if err != nil {
			PrintMe("ttt", "err", err)
			return reconcile.Result{}, err
		}
		patch, err := jsonpatch.CreatePatch(podJSON, newPodJSON)
		if err != nil {
			PrintMe("ttsst", "err", err)
			return reconcile.Result{}, err
		}
		_ = patch
		payloadBytes, _ := json.Marshal(patch)
		//PrintMe("the patch", string(payloadBytes))

		jpatch := client.ConstantPatch(types.JSONPatchType, payloadBytes)
		_ = jpatch
		//err = r.client.Patch(context.TODO(), instance, jpatch)
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			PrintMe("app update err", "err", err)
			return reconcile.Result{}, err
		}
		if triggerGuruRebalance {

			guruNames := instance.Spec.Ce.GuruNames
			guruAddresses := make([]string, len(guruNames))
			for i, n := range guruNames {
				g, ok := instance.Spec.Ce.Nodes[n]
				if !ok {
					PrintMe("TODO handlefatal problem")
					return reconcile.Result{}, err
				}
				guruAddresses[i] = g.TCPAddress
			}

			var wg = sync.WaitGroup{}
			for i, pod := range aidePods {
				add := pod.Status.PodIP
				add = add + ":8080"
				wg.Add(1)
				go func() {
					err := PostUpstreamNames(guruNames, guruAddresses, pod.Name, add)
					_ = err
					wg.Done()
				}()
				_ = i
			}
			for i, pod := range guruPods {
				add := pod.Status.PodIP
				add = add + ":8080"
				wg.Add(1)
				go func() {
					err := PostUpstreamNames(guruNames, guruAddresses, pod.Name, add)
					_ = err
					wg.Done()
				}()
				_ = i
			}
			wg.Wait()
		}
	}

	rr := reconcile.Result{}
	rr.RequeueAfter = 10 * time.Second
	rr.Requeue = true
	return rr, nil
}

// newPodForCR returns a guru pod with the same name/namespace as the cr
func newPodForCR(cr *appv1alpha1.AppService) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	podName := "guru-" + getRandomString()
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "guru",
					Image:   "gcr.io/fair-theater-238820/knotfreeserver",
					Command: []string{"/go/bin/linux_386/knotfreeiot", "--server"},
					Ports: []corev1.ContainerPort{
						{Name: "iot", ContainerPort: 8384},
						{Name: "http", ContainerPort: 8080},
					},
					Env: []corev1.EnvVar{
						{Name: "POD_NAME", Value: podName},
					},
				},
			},
		},
	}
}

func getRandomString() string {
	var tmp [16]byte
	rand.Read(tmp[:])
	return hex.EncodeToString(tmp[:])
}

// GetServerStats is(
func GetServerStats(name string, address string) (*iot.ExecutiveStats, error) {

	es := &iot.ExecutiveStats{}

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
		// running over kubectl when developing locally
		cmd := `kubectl exec ` + name + ` -- curl -s localhost:8080/api1/getstats`
		kubectl.Quiet = true
		str, err := kubectl.K8s(cmd, "")
		if err != nil {
			return es, err
		}
		err = json.Unmarshal([]byte(str), &es)

		return es, err

	}
	// when in cluster
	es = iot.GetServerStats(address)

	return es, nil

}

// PostUpstreamNames does SetUpstreamNames the hard way
func PostUpstreamNames(guruList []string, addressList []string, name string, addr string) error {

	arg := &iot.UpstreamNamesArg{}
	arg.Names = guruList
	arg.Addresses = addressList

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {

		jbytes, err := json.Marshal(arg)
		if err != nil {
			PrintMe("unreachable ?? bb")
			return err
		}

		jstr := string(jbytes)

		curlcmd := `curl --header "Content-Type: application/json" --request POST --data '` + jstr + `'  http://localhost:8080/api1/set`

		cmd := `kubectl exec ` + name + ` -- ` + curlcmd

		str, err := kubectl.K8s(cmd, "")
		_ = str
		if err != nil {
			return err
		}
		return nil

	}
	// when in cluster
	err := iot.PostUpstreamNames(guruList, addressList, addr)
	return err

}
