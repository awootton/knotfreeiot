package appservice

import (
	"context"
	"fmt"

	appv1alpha1 "github.com/awootton/knotfreeiot/knotoperator/pkg/apis/app/v1alpha1"
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
)

var log = logf.Log.WithName("controller_appservice")

/**
* atw ha ha USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
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

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
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
	fmt.Println("")

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
	// else instance is the service!
	fmt.Println("spec size", instance.Spec.Size)
	fmt.Println("status size", instance.Status.Size)
	fmt.Println("namespace is ", request.Namespace)

	namespace := request.Namespace
	//namespaced := request.NamespacedName
	//namespaced.Name = "knotfreeaide"

	// can  we get the replicas of the deployment?
	// List Deployments
	fmt.Printf("Listing deployments in namespace %q:\n", namespace)

	//itemsDeploy := &corev1.ReplicationControllerList{}
	//itemsDeploy := &corev1.ReplicationControllerList{}

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
		}
	}

	// fmt.Println("depl list len", len(itemsDeploy.Items))
	// for _, d := range itemsDeploy.Items {
	// 	fmt.Println("d item", d.GetName())
	// }

	// r.client.Get()
	// list, err := r.client.Resource(deploymentRes).Namespace(namespace).List(metav1.ListOptions{})
	// if err != nil {
	// 	panic(err)
	// }
	// for _, d := range itemsDeploy.Items {
	// 	found := false
	// 	//replicas, found, err := unstructured.NestedInt64(d., "spec", "replicas")
	// 	if err != nil || !found {
	// 		fmt.Printf("Replicas not found for deployment %s: error=%s", d.GetName(), err)
	// 		continue
	// 	}
	// 	fmt.Printf(" * %s (%d replicas)\n", d.GetName()) //, replicas)
	// }

	// fmt.Printf("Listing deployments in namespace %q:\n", request.NamespacedName)
	// list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	panic(err)
	// }
	// for _, d := range list.Items {
	// 	fmt.Printf(" * %s (%d replicas)\n", d.Name, *d.Spec.Replicas)
	// }

	// Define a new Pod object
	pod := newPodForCR(instance)

	// Set AppService instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			fmt.Println("pod create fail", err)
			return reconcile.Result{}, err
		}
		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		fmt.Println("pod get fail", err)
		return reconcile.Result{}, err
	}

	// else  Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)

	items2 := &corev1.NodeList{}

	err2 := r.client.List(context.TODO(), items2)
	_ = err2
	if err2 == nil {
		//reqLogger.Info("list found ", "list:", *items)
		for i, node := range items2.Items {
			nname := node.GetName()
			fmt.Println("node/", nname)
			_ = i
		}
	}

	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *appv1alpha1.AppService) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "knotguru",
					Image:   "gcr.io/fair-theater-238820/knotfreeserver",
					Command: []string{"/go/bin/linux_386/knotfreeiot", ""},
				},
			},
		},
	}
}
