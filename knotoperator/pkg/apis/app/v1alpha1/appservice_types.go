package v1alpha1

import (
	"github.com/awootton/knotfreeiot/iot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN! atw ok
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AppServiceSpec defines the desired state of AppService
type AppServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	//	AideCount     int      `json:"aidecount"`
	//GuruNames []string `json:"gurunames"` //this specifies an ordering
	//	GuruAddresses []string `json:"guruaddresses"`

	//nodes map[string]NodeStats // includes aides

	Ce *ClusterState
}

// ClusterState is too much like see iot.ClusterExecutive
type ClusterState struct {
	GuruNames []string `json:"gurunames"` //this specifies an ordering

	// name to stats
	Nodes map[string]*NodeStats // includes aides
}

// NewClusterState is
func NewClusterState() *ClusterState {
	ce := &ClusterState{}
	ce.GuruNames = make([]string, 0)
	ce.Nodes = make(map[string]*NodeStats)
	return ce
}

// NodeStats is too much like iot.Executive
type NodeStats struct {
	//
	Name   string
	IsGuru bool
	Stats  *iot.ExecutiveStats
}

// AppServiceStatus defines the observed state of AppService
type AppServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	//Size int `json:"size"`
	// AideCount     int      `json:"aidecount"`
	// GuruNames     []string `json:"gurunames"`
	//GuruAddresses []string `json:"guruaddresses"`
	Ce *ClusterState
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppService is the Schema for the appservices API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=appservices,scope=Namespaced
type AppService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppServiceSpec   `json:"spec,omitempty"`
	Status AppServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppServiceList contains a list of AppService
type AppServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppService{}, &AppServiceList{})
}
