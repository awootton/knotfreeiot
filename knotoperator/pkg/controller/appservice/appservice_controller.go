package appservice

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	appv1alpha1 "github.com/awootton/knotfreeiot/knotoperator/pkg/apis/app/v1alpha1"
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

var log = logf.Log.WithName("controller_appservice")

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

// Reconcile reads that state of the cluster for a AppService object and makes changes based on the state read
// and what is in the AppService.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAppService) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	fmt.Println("")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("")
	fmt.Println("count", count)
	count++

	appchanged := false

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AppService")

	// Fetch the AppService instance
	instance := &appv1alpha1.AppService{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			fmt.Println("AppService NOT FOUND")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		fmt.Println("AppService FAIL", err)
		return reconcile.Result{}, err
	}
	virginInstance := instance
	instance = instance.DeepCopy()

	// else instance is the service!
	fmt.Println("spec names", instance.Spec.GuruNames)
	fmt.Println("status names", instance.Status.GuruNames)
	fmt.Println("namespace is ", request.Namespace)
	fmt.Println("spec count", instance.Spec.AideCount)
	fmt.Println("status count", instance.Status.AideCount)

	if (len(instance.Spec.GuruNames) > 0 && instance.Spec.GuruNames[0] == "deleteme") || len(instance.Spec.GuruNames) != len(instance.Spec.GuruAddresses) {
		instance.Spec.GuruNames = instance.Spec.GuruNames[0:0]
		instance.Spec.GuruAddresses = instance.Spec.GuruAddresses[0:0]
		appchanged = true
	}

	namespace := request.Namespace

	knownNames := make(map[string]int)
	for i, name := range instance.Spec.GuruNames {
		knownNames[name] = i
	}

	items := &corev1.PodList{}

	aides := make([]corev1.Pod, 0)
	gurus := make([]corev1.Pod, 0)
	others := make([]corev1.Pod, 0)

	err2 := r.client.List(context.TODO(), items)
	_ = err2
	if err2 == nil {
		for i, pod := range items.Items {
			pname := pod.GetName()

			if strings.HasPrefix(pname, "aide-") {
				aides = append(aides, pod)
			} else if strings.HasPrefix(pname, "guru-") {
				gurus = append(aides, pod)
			} else {
				others = append(others, pod)
			}

			_ = i
		}
	} else {
		return reconcile.Result{}, err2
	}

	neededGurus := 1

	// are there any non-pending guru's that are not on the list?
	for i, pod := range gurus {
		index, present := knownNames[pod.Name]
		if present == false {
			// so it's not on the list.
			// is it running?
			ready := pod.Status.ContainerStatuses[0].Ready
			if ready && len(pod.Status.PodIP) > 0 {

				instance.Spec.GuruNames = append(instance.Spec.GuruNames, pod.Name)
				address := pod.Status.PodIP
				fmt.Println("new guru has ip ", address)
				instance.Spec.GuruAddresses = append(instance.Spec.GuruAddresses, address+":8384")
				appchanged = true

			}
		}
		_ = i
		_ = index
	}

	var wg = sync.WaitGroup{}

	for i, pod := range aides {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func() {
			es := GetServerStats(pod.Name, add)
			fmt.Println(es)
			wg.Done()
		}()
		_ = i
	}
	for i, pod := range gurus {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func() {
			es := GetServerStats(pod.Name, add)
			fmt.Println(es)
			wg.Done()
		}()
		_ = i
	}
	wg.Wait()

	if len(gurus) < neededGurus {
		// make a new one
		pod := newPodForCR(instance)
		// Set AppService instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			fmt.Println("SetControllerReference ", err)
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			fmt.Println("pod create fail", err)
			return reconcile.Result{}, err
		}
		// don't modify the app yet.
		//instance.Spec.GuruNames = append(instance.Spec.GuruNames, pod.Name)
		//address := pod.Status.PodIP
		//fmt.Println("new guru has ip ", address)
		//instance.Spec.GuruAddresses = append(instance.Spec.GuruAddresses, address+":8384")
		//appchanged = true
	}

	items2 := &corev1.NodeList{}

	err3 := r.client.List(context.TODO(), items2)
	if err3 == nil {
		for i, node := range items2.Items {
			nname := node.GetName()
			fmt.Println("node/", nname)
			_ = i
		}
	}

	// can  we get the replicas of the deployment?
	// List Deployments
	fmt.Printf("Listing deployments in namespace %q:\n", namespace)
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
		fmt.Println("depl list get err", geterr)
		return reconcile.Result{}, err
	}
	name, ok, err := unstructured.NestedString(u.UnstructuredContent(), "metadata", "name")
	fmt.Println("get got name ", name, ok, err)

	repcount, ok, err := unstructured.NestedInt64(u.UnstructuredContent(), "spec", "replicas")
	fmt.Println("get got repcount ", repcount, ok, err)

	targetRepCount := int64(1)

	if geterr == nil && repcount != targetRepCount {
		unstructured.SetNestedField(u.UnstructuredContent(), targetRepCount, "spec", "replicas")
		err = r.client.Update(context.TODO(), u)
		if err != nil {
			fmt.Println("update err", err)
			return reconcile.Result{}, err
		}
	}

	if appchanged {
		fmt.Println("UPDATING app")
		fmt.Println("UPDATING app")
		fmt.Println("UPDATING app")

		podJSON, err := json.Marshal(virginInstance)
		if err != nil {
			fmt.Println("sss", err)
			return reconcile.Result{}, err
		}
		newPodJSON, err := json.Marshal(instance)
		if err != nil {
			fmt.Println("ttt", err)
			return reconcile.Result{}, err
		}
		patch, err := jsonpatch.CreatePatch(podJSON, newPodJSON)
		if err != nil {
			fmt.Println("ttsst", err)
			return reconcile.Result{}, err
		}
		_ = patch
		payloadBytes, _ := json.Marshal(patch)
		//fmt.Println("the patch", string(payloadBytes))

		jpatch := client.ConstantPatch(types.JSONPatchType, payloadBytes)

		err = r.client.Patch(context.TODO(), instance, jpatch)
		//err = r.client.Update(context.TODO(), instance)
		if err != nil {
			fmt.Println("app update err", err)
			return reconcile.Result{}, err
		}

		var wg = sync.WaitGroup{}
		for i, pod := range aides {
			add := pod.Status.PodIP
			add = add + ":8080"
			wg.Add(1)
			go func() {
				es := PostUpstreamNames(instance.Spec.GuruNames, instance.Spec.GuruAddresses, add)
				fmt.Println(es)
				wg.Done()
			}()
			_ = i
		}
		for i, pod := range gurus {
			add := pod.Status.PodIP
			add = add + ":8080"
			wg.Add(1)
			go func() {
				es := PostUpstreamNames(instance.Spec.GuruNames, instance.Spec.GuruAddresses, add)
				fmt.Println(es)
				wg.Done()
			}()
			_ = i
		}
		wg.Wait()

	}

	return reconcile.Result{}, nil
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

// GetServerStats is
func GetServerStats(name string, address string) *iot.ExecutiveStats {

	es := &iot.ExecutiveStats{}

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
		// running over kubectl when developing locally
		cmd := `kubectl exec ` + name + ` -- curl -s localhost:8080/api1/getstats`
		//fmt.Println(cmd)
		str, err := K8s(cmd, "")
		//fmt.Println(str)
		err = json.Unmarshal([]byte(str), &es)
		_ = err

	} else {
		// when in cluster
		es = iot.GetServerStats(address)
	}

	return es

}

// PostUpstreamNames does SetUpstreamNames the hard way
func PostUpstreamNames(guruList []string, addressList []string, addr string) error {

	arg := &iot.UpstreamNamesArg{}
	arg.Names = guruList
	arg.Addresses = addressList

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {

		jbytes, err := json.Marshal(arg)
		if err != nil {
			fmt.Println("unreachable ?? bb")
			return err
		}

		resp, err := http.Post("http://"+addr+"/api1/set", "application/json", bytes.NewReader(jbytes))
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return &errors.StatusError{}
		}
		return nil

	}
	// when in cluster
	err := iot.PostUpstreamNames(guruList, addressList, addr)
	return err

}
