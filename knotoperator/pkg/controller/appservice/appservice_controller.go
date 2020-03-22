package appservice

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/knotoperator/pkg/apis/app/v1alpha1"
	appv1alpha1 "github.com/awootton/knotfreeiot/knotoperator/pkg/apis/app/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
////var LastClusterState *appv1alpha1.ClusterState

// this doesn't work :: var reconcileLogger logr.Logger = log

// Reconcile reads that state of the cluster for a AppService object and makes changes based on the state read
// and what is in the AppService.Spec
// Note: fixme too long
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAppService) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues() //"Request.Namespace", request.Namespace, "Request.Name", request.Name)
	//reconcileLogger = reqLogger
	reqLogger.Info("Reconciling AppService")

	PrintMe := func(msg string, args ...interface{}) {
		if reqLogger != nil {
			if len(args)&1 == 0 { // doesn't work
				reqLogger.Info(msg, "args", args)
			} else {
				reqLogger.Info(msg, "args", args)
			}
		}
	}

	PrintMe("count", count) // atw fixme: this is a stat, not a log
	count++

	rr := reconcile.Result{}
	rr.RequeueAfter = 5 * time.Second
	rr.Requeue = true

	// Fetch the AppService instance
	gotinstance := &appv1alpha1.AppService{}
	err := r.client.Get(context.TODO(), request.NamespacedName, gotinstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			PrintMe("AppService NOT FOUND") // atw fixme: this is an error, not a log
			return rr, nil
		}
		// Error reading the object - requeue the request.
		PrintMe("AppService FAIL", err)
		return rr, err
	}
	virginInstance := gotinstance
	workingInstance := gotinstance.DeepCopy()

	s := &status{}

	s.instance = workingInstance
	s.r = r
	s.appchangedReasons = make([]string, 0)
	s.triggerGuruRebalanceReasons = make([]string, 0)
	s.aidePods = make(map[string]corev1.Pod, 0)
	s.guruPods = make(map[string]corev1.Pod, 0)
	s.otherPods = make(map[string]corev1.Pod, 0)
	s.aidesPending = make(map[string]corev1.Pod, 0)
	s.gurusPending = make(map[string]corev1.Pod, 0)
	s.PrintMe = PrintMe

	if s.instance.Status.Ce == nil {
		s.instance.Status.Ce = v1alpha1.NewClusterState()
	} else if len(s.instance.Status.Ce.GuruNames) == 0 {
		s.instance.Status.Ce = v1alpha1.NewClusterState()
		s.appChanged("empty status GuruNames")
	}
	if s.instance.Spec.Ce == nil {
		s.instance.Spec.Ce = v1alpha1.NewClusterState()
		s.appChanged("empty spec")
	} else if len(s.instance.Spec.Ce.GuruNames) == 0 {
		//s.instance.Spec.Ce = v1alpha1.NewClusterState()
		//s.appChanged("empty spec GuruNames")
	}
	if len(s.appchangedReasons) != 0 {
		s.triggerGuruRebalance("app changed")
	}

	//LastClusterState = s.instance.Spec.Ce

	//PrintMe("guru ", "names", instance.Spec.Ce.GuruNames)
	//namespace := request.Namespace

	items := &corev1.PodList{}
	err2 := r.client.List(context.TODO(), items)
	if err2 != nil {
		return rr, err2
	}

	// load the current state into s
	s.loadPodList(items)

	// http to all the nodes (aides and gurus) and get their status
	// add prom timer here
	s.httpGetStatusAllPods()

	// so now we have all the stats
	// add prom timer here
	s.httpSendAllStatusToAllPods()

	// so now we have all the stats
	aideList := make([]*iot.ExecutiveStats, 0)
	guruList := make([]*iot.ExecutiveStats, 0)

	for key, val := range s.instance.Spec.Ce.Nodes {
		if strings.HasPrefix(val.Name, "aide-") {
			aideList = append(aideList, val)
		} else {
			guruList = append(guruList, val)
		}
		_ = key
	}

	resize := iot.CalcExpansionDesired(aideList, guruList)

	if len(guruList) > 7 || len(s.gurusPending) != 0 { //FIXME: remove this
		if resize.ChangeGurus == 1 {
			resize.ChangeGurus = 0 //FIXME: remove this
		}
	}

	if s.instance.Spec.Ce.GuruNamesPending != 0 && s.instance.Spec.Ce.GuruNamesPending+10 < uint32(time.Now().Unix()) {
		s.instance.Spec.Ce.GuruNamesPending = 0
		s.appChanged("timed out GuruNamesPending")
	}

	if resize.ChangeGurus > 0 || (len(guruList) == 0 && 0 == len(s.gurusPending) && s.instance.Spec.Ce.GuruNamesPending == 0) { //} len(gurus) < neededGurus {

		PrintMe("ADDING GURU")

		// make a new one
		pod := newPodForCR(s.instance)
		//s.triggerGuruRebalance("adding guru" + pod.GetName()) do later when it's online

		// Set AppService instance as the owner and controller
		if err := controllerutil.SetControllerReference(s.instance, pod, r.scheme); err != nil {
			PrintMe("SetControllerReference ", "err", err)
			return rr, err
		}
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			PrintMe("pod create fail", "err", err)
			return rr, err
		}
		s.instance.Spec.Ce.GuruNamesPending = uint32(time.Now().Unix())
		s.appChanged("guru is pending")

	} else if resize.ChangeGurus < 0 {

		PrintMe("DELETING GURU")

		// delete the last one
		name := s.instance.Spec.Ce.GuruNames[len(s.instance.Spec.Ce.GuruNames)-1]
		pod, ok := s.guruPods[name]
		if ok {
			reqLogger.Info("Deleting a Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", name)
			err := r.client.Delete(context.TODO(), &pod)
			if err != nil {
				PrintMe("pod delete fail", "err", err)
				return rr, err
			}
		}
		s.mux.Lock()
		s.instance.Spec.Ce.GuruNames = s.instance.Spec.Ce.GuruNames[0 : len(s.instance.Spec.Ce.GuruNames)-1]
		delete(s.instance.Spec.Ce.Nodes, name)
		s.appChanged("deleted guru " + name)
		s.triggerGuruRebalance("deleted guru" + pod.GetName())

		s.mux.Unlock()
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "Deployment",
		Version: "v1",
	})

	targetRepCount, geterr := s.getCurrentAidesReplCount(u)
	repcount := targetRepCount

	if geterr != nil {
		return rr, err
	}

	if resize.ChangeAides > 0 && len(s.aidesPending) == 0 {
		targetRepCount++
		reqLogger.Info("More aides")
	} else if resize.ChangeAides < 0 || len(s.aidesPending) > 2 {
		targetRepCount--
	}
	if targetRepCount < 2 {
		targetRepCount = 2
	}

	if geterr == nil && repcount != targetRepCount {
		unstructured.SetNestedField(u.UnstructuredContent(), targetRepCount, "spec", "replicas")
		err = r.client.Update(context.TODO(), u)
		if err != nil {
			PrintMe("update err", "err", err)
			return rr, err
		}
	}

	if len(s.triggerGuruRebalanceReasons) != 0 {

		PrintMe("REBALANCE gurus because: ", s.appchangedReasons)

		err := s.rebalanceGurus()
		if err != nil {
			return rr, err
		}
	}

	if len(s.appchangedReasons) != 0 {

		PrintMe("UPDATING app because: ", s.appchangedReasons)
		err := s.updateApp(virginInstance)
		if err != nil {
			PrintMe("UPDATING app error", err)
			return rr, err
		}
	}

	rr.RequeueAfter = 11 * time.Second
	rr.Requeue = true
	return rr, nil
}

// newPodForCR returns a guru pod with the same name/namespace as the cr
func newPodForCR(cr *appv1alpha1.AppService) *corev1.Pod {

	registry := "gcr.io/fair-theater-238820"
	r, ok := os.LookupEnv("REGISTRY")
	if ok {
		registry = r
	}
	labels := map[string]string{
		"app": cr.Name,
	}
	annots := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "9102",
	}
	podName := "guru-" + getRandomString()
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: annots,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "guru",
					Image:           registry + "/knotfreeserver",
					ImagePullPolicy: corev1.PullAlways,
					Command:         []string{"/go/bin/linux_386/knotfreeiot", "-isguru"},
					Ports: []corev1.ContainerPort{
						{Name: "iot", ContainerPort: 8384},
						{Name: "httplocal", ContainerPort: 8080},
						{Name: "prom", ContainerPort: 9102},
					},
					Env: []corev1.EnvVar{
						{Name: "POD_NAME", Value: podName},
						{Name: "MY_POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "foo", MountPath: "/root/atw/", ReadOnly: true},
					},
				},
			},
			Volumes: []corev1.Volume{
				{Name: "foo",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: "privatekeys4"},
					},
				},
			},
		},
	}
}

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

	//  (user): Modify this to be the types you create that are owned by the primary resource
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
