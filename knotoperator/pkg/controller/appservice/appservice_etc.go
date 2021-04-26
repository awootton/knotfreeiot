// // Copyright 2019,2020,2021 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package appservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	appv1alpha1 "github.com/awootton/knotfreeiot/knotoperator/pkg/apis/app/v1alpha1"
	"github.com/awootton/knotfreeiot/kubectl"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type status struct {
	appchangedReasons           []string
	triggerGuruRebalanceReasons []string

	instance *appv1alpha1.AppService
	//	items     *corev1.PodList
	aidePods  map[string]corev1.Pod
	guruPods  map[string]corev1.Pod
	otherPods map[string]corev1.Pod

	aidesPending map[string]corev1.Pod
	gurusPending map[string]corev1.Pod

	mux sync.Mutex

	r *ReconcileAppService

	PrintMe func(msg string, args ...interface{})
}

func (s *status) rebalanceGurus() error {

	if len(s.instance.Spec.Ce.GuruNames) == 0 {
		// why are we here? The aides reject zero len upgrades.
		//PrintMe("zero len GuruNames")
		return nil // errors.New("zero len GuruNames")
	}

	guruNames := s.instance.Spec.Ce.GuruNames
	guruAddresses := make([]string, len(guruNames))
	for i, n := range guruNames {
		g, ok := s.instance.Spec.Ce.Nodes[n]
		if !ok {
			s.PrintMe("no stat for name", n, s.instance.Spec.Ce.Nodes)
			return errors.New(fmt.Sprint("no stat for name", n, s.instance.Spec.Ce.Nodes))
		}
		guruAddresses[i] = g.TCPAddress
	}

	errs := make([]error, 0)

	var wg = sync.WaitGroup{}
	for i, pod := range s.aidePods {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func() {
			err := postUpstreamNames(guruNames, guruAddresses, pod.Name, add)
			if err != nil {
				errs = append(errs, err)
			}
			wg.Done()
		}()
		_ = i
	}
	for i, pod := range s.guruPods {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func() {
			err := postUpstreamNames(guruNames, guruAddresses, pod.Name, add)
			if err != nil {
				errs = append(errs, err)
			}
			wg.Done()
		}()
		_ = i
	}
	wg.Wait()

	if len(errs) != 0 {
		return errors.New(fmt.Sprint("postUpstreamNames errors", errs))
	}

	return nil
}

func (s *status) updateApp(virginInstance *appv1alpha1.AppService) error {

	s.PrintMe("UPDATING app because: ")

	podJSON, err := json.Marshal(virginInstance)
	if err != nil {
		s.PrintMe("sss", "err", err)
		return err
	}
	newPodJSON, err := json.Marshal(s.instance)
	if err != nil {
		s.PrintMe("ttt", "err", err)
		return err
	}
	patch, err := jsonpatch.CreatePatch(podJSON, newPodJSON)
	if err != nil {
		s.PrintMe("ttsst", "err", err)
		return err
	}
	_ = patch
	payloadBytes, _ := json.Marshal(patch)
	//s.PrintMe("the patch", string(payloadBytes))

	jpatch := client.ConstantPatch(types.JSONPatchType, payloadBytes)
	_ = jpatch
	//err = r.client.Patch(context.TODO(), s.instance, jpatch)
	err = s.r.client.Update(context.TODO(), s.instance)
	if err != nil {
		s.PrintMe("app update err", "err", err)
		return err
	}
	return nil
}

func (s *status) getCurrentAidesReplCount(u *unstructured.Unstructured) (int64, error) {
	// can  we get the replicas of the deployment?
	// List Deployments
	// Using a unstructured object.

	ckey := client.ObjectKey{
		Namespace: "knotspace",
		Name:      "aide",
	}
	geterr := s.r.client.Get(context.TODO(), ckey, u)
	if geterr != nil {
		s.PrintMe("depl list get ", "err", geterr)
		return 0, geterr //reconcile.Result{}, geterr
	}
	name, ok, err := unstructured.NestedString(u.UnstructuredContent(), "metadata", "name")
	//s.PrintMe("get got name ", name, ok, err)
	if err != nil {
		s.PrintMe("get meta name", "err", geterr)
		return 0, err //reconcile.Result{}, geterr
	}
	_ = name

	repcount, ok, err := unstructured.NestedInt64(u.UnstructuredContent(), "spec", "replicas")
	//s.PrintMe("get got repcount ", repcount, ok, err)

	if !ok {
		str := "get spec replicas"
		s.PrintMe(str, "err", geterr)
		return 0, errors.New(str)
	}

	//	targetRepCount := int64(len(s.aidePods))
	return repcount, nil
}

func (s *status) httpSendAllStatusToAllPods() {
	// so now we have all the stats
	{ // tell everyone
		stats := make([]*iot.ExecutiveStats, 0, len(s.instance.Spec.Ce.Nodes))
		for _, val := range s.instance.Spec.Ce.Nodes {
			stats = append(stats, val)
		}
		var wg2 = sync.WaitGroup{}
		for _, s := range stats {
			wg2.Add(1)
			go func(s *iot.ExecutiveStats) {
				err := postClusterStats(stats, s.Name, s.HTTPAddress)
				if err != nil {
					fmt.Println("posting cluster stats ", err)
				}
				wg2.Done()
			}(s)
		}
		wg2.Wait()
	}
}

func (s *status) httpGetStatusAllPods() {
	// http to all the nodes (aides and gurus) and get their status
	var wg = sync.WaitGroup{}

	for i, pod := range s.aidePods {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func(name, add string) {
			defer wg.Done()
			es, err := getServerStats(name, add)
			s.mux.Lock()
			defer s.mux.Unlock()
			if err != nil {
				delete(s.instance.Spec.Ce.Nodes, name)
			} else {
				nodeStats, present := s.instance.Spec.Ce.Nodes[name]
				if present {
					es.TCPAddress = nodeStats.TCPAddress
					es.HTTPAddress = nodeStats.HTTPAddress
					if !reflect.DeepEqual(nodeStats, es) {
						s.instance.Spec.Ce.Nodes[name] = es
						//appchanged()
					}
				}
			}
		}(pod.Name, add)
		_ = i
	}
	for i, pod := range s.guruPods {
		add := pod.Status.PodIP
		add = add + ":8080"
		wg.Add(1)
		go func(name, add string) {
			defer wg.Done()
			es, err := getServerStats(name, add)
			s.mux.Lock()
			defer s.mux.Unlock()
			if err != nil {
				delete(s.instance.Spec.Ce.Nodes, name)
			} else {
				nodeStats, present := s.instance.Spec.Ce.Nodes[name]
				if present {
					es.TCPAddress = nodeStats.TCPAddress
					es.HTTPAddress = nodeStats.HTTPAddress
					if !reflect.DeepEqual(nodeStats, es) {
						s.instance.Spec.Ce.Nodes[name] = es
						//appchanged()
					}
				}
			}
		}(pod.Name, add)
		_ = i
	}
	wg.Wait()

}

func (s *status) appChanged(why string) {
	s.appchangedReasons = append(s.appchangedReasons, why)
}

func (s *status) triggerGuruRebalance(why string) {
	s.triggerGuruRebalanceReasons = append(s.triggerGuruRebalanceReasons, why)
}

func (s *status) loadPodList(items *corev1.PodList) {

	for i, pod := range items.Items {
		pname := pod.GetName()

		if strings.HasPrefix(pname, "aide-") {
			s.aidePods[pname] = pod
		} else if strings.HasPrefix(pname, "guru-") {
			s.guruPods[pname] = pod
		} else {
			s.otherPods[pname] = pod
		}
		_ = i
	}

	// are there nodes on the AppService list that don't exist as pods?
	for name, spec := range s.instance.Spec.Ce.Nodes {
		if strings.HasPrefix(name, "aide-") {
			_, ok := s.aidePods[name]
			if ok == false {
				delete(s.instance.Spec.Ce.Nodes, name)
				s.appChanged("aide deleted from official list")
			}
		} else if strings.HasPrefix(name, "guru-") {
			_, ok := s.guruPods[name]
			if ok == false {
				delete(s.instance.Spec.Ce.Nodes, name)
				s.appChanged("guru deleted from official list")
				s.triggerGuruRebalance("guru deleted from official list")
				for i, str := range s.instance.Spec.Ce.GuruNames {
					if str == name {
						// remove from GuruNames also which creates a big rebalance.
						lm1 := len(s.instance.Spec.Ce.GuruNames) - 1
						s.instance.Spec.Ce.GuruNames[i] = s.instance.Spec.Ce.GuruNames[lm1]
						s.instance.Spec.Ce.GuruNames = s.instance.Spec.Ce.GuruNames[0:lm1]
						break
					}
				}
			}
		}
		_ = spec
	}

	// clean up guru names on the AppService list
	for _, name := range s.instance.Spec.Ce.GuruNames {
		if strings.HasPrefix(name, "aide-") {
			s.PrintMe("HOW did an AIDE get on this list?")
		} else if strings.HasPrefix(name, "guru-") {
			_, ok := s.guruPods[name]
			if ok == false {
				delete(s.instance.Spec.Ce.Nodes, name)
				s.appChanged("guru deleted from nodes array")
				s.triggerGuruRebalance("guru deleted from nodes array")
				for i, str := range s.instance.Spec.Ce.GuruNames {
					if str == name {
						// remove from GuruNames also which creates a big rebalance.
						lm1 := len(s.instance.Spec.Ce.GuruNames) - 1
						s.instance.Spec.Ce.GuruNames[i] = s.instance.Spec.Ce.GuruNames[lm1]
						s.instance.Spec.Ce.GuruNames = s.instance.Spec.Ce.GuruNames[0:lm1]
						break
					}
				}
			}
		}
	}

	// are there any non-pending guru's that are not on the AppService list?
	// what if it's on the list but it's not feeling well?
	for i, pod := range s.guruPods {
		nodeStats, present := s.instance.Spec.Ce.Nodes[pod.Name]
		if present == false {
			// so it's not on the list.
			// is it running?
			ready := false
			if len(pod.Status.ContainerStatuses) != 0 {
				ready = pod.Status.ContainerStatuses[0].Ready
			}
			if ready && len(pod.Status.PodIP) > 0 {
				// if it's ready now and not on the list then make it official.

				stats := new(iot.ExecutiveStats)
				stats.TCPAddress = pod.Status.PodIP + ":8384"
				stats.HTTPAddress = pod.Status.PodIP + ":8080"
				stats.Name = pod.Name

				s.instance.Spec.Ce.Nodes[pod.Name] = stats

				s.instance.Spec.Ce.GuruNames = append(s.instance.Spec.Ce.GuruNames, pod.Name)

				s.appChanged("added new ready guru to nodes array ")
				s.triggerGuruRebalance("added new ready guru to nodes array ")
				s.instance.Spec.Ce.GuruNamesPending = 0

			} else {

				s.gurusPending[pod.Name] = pod

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
	for i, pod := range s.aidePods {
		nodeStats, present := s.instance.Spec.Ce.Nodes[pod.Name]
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
				s.instance.Spec.Ce.Nodes[pod.Name] = stats

				s.appChanged("add ready aide to nodes")
				s.triggerGuruRebalance("add ready aide to nodes") // FIXME: We really only need to send the gurunames to this pod not everyone.
			} else {
				s.aidesPending[pod.Name] = pod
			}
		} else if len(pod.Status.ContainerStatuses) != 0 {
			// we found it.
			// update the address
			if pod.Status.ContainerStatuses[0].Ready && len(pod.Status.PodIP) > 0 {
				nodeStats.TCPAddress = pod.Status.PodIP + ":8384"
				nodeStats.HTTPAddress = pod.Status.PodIP + ":8080"
			}
		}
		_ = i
	}

}

func getRandomString() string {
	var tmp [16]byte
	rand.Read(tmp[:])
	return hex.EncodeToString(tmp[:])
}

// getServerStats is
func getServerStats(name string, address string) (*iot.ExecutiveStats, error) {

	es := &iot.ExecutiveStats{}

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {
		// running over kubectl when developing locally
		cmd := `kubectl exec ` + name + ` -- curl -s localhost:8080/api2/getstats`
		kubectl.Quiet = true
		kubectl.SuperQuiet = true
		str, err := kubectl.K8s(cmd, "")
		if err != nil {
			return es, err
		}
		err = json.Unmarshal([]byte(str), &es)

		return es, err

	}
	// when in cluster
	var err error
	es, err = iot.GetServerStats(address)

	return es, err

}

// postClusterStats  post ClusterStats
func postClusterStats(array []*iot.ExecutiveStats, name string, addr string) error {

	stats := &iot.ClusterStats{}
	stats.When = uint32(time.Now().Unix())
	stats.Stats = array

	// fmt.Println("starting PostClusterStats with ")
	// start := time.Now()
	// defer func() {
	// 	end := time.Now()
	// 	dur := end.Sub(start)
	// 	fmt.Println("time PostClusterStats = ", dur)
	// }()

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {

		jbytes, err := json.Marshal(stats)
		if err != nil {
			//PrintMe("unreachable ?? bb")
			return err
		}

		jstr := string(jbytes)

		curlcmd := `curl -s --header "Content-Type: application/json" --request POST --data '` + jstr + `'  http://localhost:8080/api2/clusterstats`

		cmd := `kubectl exec ` + name + ` -- ` + curlcmd
		kubectl.Quiet = true
		kubectl.SuperQuiet = true
		str, err := kubectl.K8s(cmd, "")
		_ = str
		if err != nil {
			return err
		}
		return nil

	}
	// when in cluster

	err := iot.PostClusterStats(stats, addr)

	return err

}

// postUpstreamNames does SetUpstreamNames the hard way
func postUpstreamNames(guruList []string, addressList []string, name string, addr string) error {

	arg := &iot.UpstreamNamesArg{}
	arg.Names = guruList
	arg.Addresses = addressList

	fmt.Println("starting names post with ", guruList, " to ", addr, name)
	start := time.Now()

	defer func() {
		end := time.Now()
		dur := end.Sub(start)
		fmt.Println("time names post = ", dur)
	}()

	if os.Getenv("KUBE_EDITOR") == "atom --wait" {

		jbytes, err := json.Marshal(arg)
		if err != nil {
			//PrintMe("unreachable ?? bb")
			return err
		}

		jstr := string(jbytes)

		curlcmd := `curl -s --header "Content-Type: application/json" --request POST --data '` + jstr + `'  http://localhost:8080/api2/set`

		cmd := `kubectl exec ` + name + ` -- ` + curlcmd
		kubectl.Quiet = true
		kubectl.SuperQuiet = true
		str, err := kubectl.K8s(cmd, "")
		_ = str
		if err != nil {
			return err
		}
		return nil

	}
	// when in cluster

	err := iot.PostUpstreamNames(guruList, addressList, addr)
	if err != nil {
		fmt.Println("PostUpstreamNames fail", addr, err)
	} else {
		fmt.Println("PostUpstreamNames to", addr, guruList)
	}

	return err

}
