/*


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

package v1alpha4

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
const (
	DataSecretAvailableCondition clusterv1.ConditionType = "DataSecretAvailable"
)

// EtcdadmConfigSpec defines the desired state of EtcdadmConfig
type EtcdadmConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of EtcdadmConfig. Edit EtcdadmConfig_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// EtcdadmConfigStatus defines the observed state of EtcdadmConfig
type EtcdadmConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Conditions defines current service state of the KubeadmConfig.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	DataSecretName *string `json:"dataSecretName,omitempty"`

	Ready bool `json:"ready,omitempty"`
}

func (c *EtcdadmConfig) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *EtcdadmConfig) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// EtcdadmConfig is the Schema for the etcdadmconfigs API
type EtcdadmConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtcdadmConfigSpec   `json:"spec,omitempty"`
	Status EtcdadmConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EtcdadmConfigList contains a list of EtcdadmConfig
type EtcdadmConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EtcdadmConfig `json:"items"`
}

// +kubebuilder:object:root=true

// EtcdInitConfiguration contains config options for etcdadm init command
type EtcdInitConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EtcdadmConfig{}, &EtcdadmConfigList{})
}
