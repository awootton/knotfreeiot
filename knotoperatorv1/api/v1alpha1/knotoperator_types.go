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

package v1alpha1

import (
	"github.com/awootton/knotfreeiot/iot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SHMEDIT THIS FILE! ok ok ok already atw THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KnotoperatorSpec defines the desired state of Knotoperator
type KnotoperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Knotoperator. Edit knotoperator_types.go to remove/update
	//Foo string `json:"foo,omitempty"`

	Ce *ClusterState `json:"ce,omitempty"`
}

// ClusterState is too much like see iot.ClusterExecutive
type ClusterState struct {
	GuruNames []string `json:"gurunames"` //this specifies an ordering
	// name to stats
	Nodes map[string]*iot.ExecutiveStats `json:"nodes"` // includes aides

	GuruNamesPending uint32 `json:"guruNamesPending"` // when we started last guru pod or zero if none
}

// KnotoperatorStatus defines the observed state of Knotoperator
type KnotoperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// atw *** -->> Important: Run "make" to regenerate code after modifying this file

	Ce *ClusterState `json:"ce,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Knotoperator is the Schema for the knotoperators API
// +kubebuilder:subresource:status
type Knotoperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnotoperatorSpec   `json:"spec,omitempty"`
	Status KnotoperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KnotoperatorList contains a list of Knotoperator
type KnotoperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Knotoperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Knotoperator{}, &KnotoperatorList{})
}
