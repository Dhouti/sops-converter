/*
Copyright © 2020 Rex Via  l.rex.via@gmail.com

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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SopsSecretStatus defines the observed state of SopsSecret
type SopsSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// SopsSecret is the Schema for the sopssecrets API
type SopsSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Type   corev1.SecretType `json:"type,omitempty"`
	Spec   SopsSecretSpec    `json:"spec,omitempty"`
	Data   string            `json:"data,omitempty"`
	Status SopsSecretStatus  `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SopsSecretList contains a list of SopsSecret
type SopsSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SopsSecret `json:"items"`
}

type SopsSecretSpec struct {
	Template       SopsSecretTemplate `json:"template,omitempty"`
	IgnoredKeys    []string           `json:"ignoredKeys,omitempty"`
	SkipFinalizers bool               `json:"skipFinalizers,omitempty"`
}

type SopsSecretTemplate struct {
	SopsSecretTemplateMetadata `json:"metadata,omitempty"`
}

type SopsSecretTemplateMetadata struct {
	Name       string   `json:"name,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SopsSecret{}, &SopsSecretList{})
}
