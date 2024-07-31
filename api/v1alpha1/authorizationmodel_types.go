/*
Copyright 2024.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthorizationModel defines the desired state of AuthorizationModel
type AuthorizationModelSpec struct {
	StoreRef corev1.LocalObjectReference `json:"storeRef"`
	Model    string                      `json:"model,omitempty"`
}

// AuthorizationModelStatus defines the observed state of AuthorizationModel
type AuthorizationModelStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AuthorizationModel is the Schema for the AuthorizationModels API
type AuthorizationModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthorizationModelSpec   `json:"spec,omitempty"`
	Status AuthorizationModelStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AuthorizationModelList contains a list of AuthorizationModel
type AuthorizationModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthorizationModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthorizationModel{}, &AuthorizationModelList{})
}

func (i *AuthorizationModel) GetConditions() []metav1.Condition { return i.Status.Conditions }
func (i *AuthorizationModel) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}