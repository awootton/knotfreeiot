/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/awootton/knotfreeiot/iot"

	cachev1alpha1 "github.com/awootton/knotfreeiot/knotoperatorv1/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KnotoperatorReconciler reconciles a Knotoperator object
type KnotoperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var count int

//+kubebuilder:rbac:groups=cache.knotfree.net,resources=knotoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.knotfree.net,resources=knotoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.knotfree.net,resources=knotoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Knotoperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile

// +kubebuilder:rbac:groups=cache.knotfree.net,resources=knotoperators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.knotfree.net,resources=knotoperators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.knotfree.net,resources=knotoperators/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
func (r *KnotoperatorReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	reqLogger := log.FromContext(ctx) //.WithValues() //"Request.Namespace", request.Namespace, "Request.Name", request.Name)
	//reconcileLogger = reqLogger
	reqLogger.Info("Reconciling AppService")

	PrintMe := func(msg string, args ...interface{}) {
		//if reqLogger != nil {
		//	if len(args)&1 == 0 { // doesn't work
		reqLogger.Info(msg, "args", args)
		//	} else {
		//		reqLogger.Info(msg, "args", args)
		//	}
		//}
	}

	PrintMe("count", count) // atw fixme: this is a stat, not a log
	count++

	rr := reconcile.Result{}
	rr.RequeueAfter = 5 * time.Second
	rr.Requeue = true

	// Fetch the AppService instance
	gotinstance := &cachev1alpha1.Knotoperator{} // appv1alpha1.AppService{}
	//err := r.client.Get(context.TODO(), request.NamespacedName, gotinstance)
	err := r.Client.Get(context.TODO(), request.NamespacedName, gotinstance)
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

	s := new(status)

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
		s.instance.Status.Ce = &cachev1alpha1.ClusterState{} // v1alpha1.NewClusterState()
		s.instance.Status.Ce.GuruNames = make([]string, 0)
		s.instance.Status.Ce.Nodes = make(map[string]*iot.ExecutiveStats, 0)
	} else if len(s.instance.Status.Ce.GuruNames) == 0 {
		s.instance.Status.Ce = &cachev1alpha1.ClusterState{} //v1alpha1.NewClusterState()
		s.appChanged("empty status GuruNames")
	}
	if s.instance.Spec.Ce == nil {
		s.instance.Spec.Ce = &cachev1alpha1.ClusterState{} //v1alpha1.NewClusterState()
		s.instance.Spec.Ce.GuruNames = make([]string, 0)
		s.instance.Spec.Ce.Nodes = make(map[string]*iot.ExecutiveStats, 0)

		s.appChanged("empty spec")
	} else if len(s.instance.Spec.Ce.GuruNames) == 0 {
		//s.instance.Spec.Ce = v1alpha1.NewClusterState()
		//s.appChanged("empty spec GuruNames")
	}
	if len(s.appchangedReasons) != 0 {
		s.triggerGuruRebalance("app changed")
	}

	//LastClusterState = s.instance.Spec.Ce

	PrintMe("guru ", "names", s.instance.Spec.Ce.GuruNames)
	//namespace := request.Namespace

	items := &corev1.PodList{}
	items.Items = make([]corev1.Pod, 0)
	err2 := r.Client.List(context.TODO(), items)
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

	PrintMe("len(guruList) len(s.gurusPending) ", len(guruList), " ", len(s.gurusPending))
	if len(guruList) > 7 || len(s.gurusPending) != 0 { //FIXME: remove this
		if resize.ChangeGurus == 1 {
			resize.ChangeGurus = 0 //FIXME: remove this
		}
	}
	PrintMe("resize.ChangeGurus ", resize.ChangeGurus)

	if s.instance.Spec.Ce.GuruNamesPending != 0 && s.instance.Spec.Ce.GuruNamesPending+10 < uint32(time.Now().Unix()) {
		s.instance.Spec.Ce.GuruNamesPending = 0
		s.appChanged("timed out GuruNamesPending")
	}

	if resize.ChangeGurus > 0 || (len(guruList) == 0 && 0 == len(s.gurusPending) && s.instance.Spec.Ce.GuruNamesPending == 0) { //} len(gurus) < neededGurus {

		PrintMe("ADDING GURU")

		// make a new one
		pod := newPodForCR(s.instance)
		//s.triggerGuruRebalance("adding guru" + pod.GetName()) do later when it's online

		// Set AppService instance as the owner and controller  r.scheme??
		if err := controllerutil.SetControllerReference(s.instance, pod, r.Scheme); err != nil {
			PrintMe("SetControllerReference ", "err", err)
			return rr, err
		}
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.Client.Create(context.TODO(), pod)
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
			err := r.Client.Delete(context.TODO(), &pod)
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
		//err = r.client.Update(context.TODO(), u)
		err = r.Client.Update(context.TODO(), u)
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

	// return ctrl.Result{}, nil
}

// newPodForCR returns a guru pod with the same name/namespace as the cr
func newPodForCR(cr *cachev1alpha1.Knotoperator) *corev1.Pod {

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

// SetupWithManager sets up the controller with the Manager.
func (r *KnotoperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Knotoperator{}).
		Complete(r)
}
