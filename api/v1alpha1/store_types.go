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

type StoreSpec struct {
	CoreModule corev1.LocalObjectReference `json:"coreModule"`
}

type StoreStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	StoreID            string             `json:"storeId,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	NextReconcileTime  metav1.Time        `json:"nextReconcileTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Store is the Schema for the stores API
type Store struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StoreSpec   `json:"spec,omitempty"`
	Status StoreStatus `json:"status,omitempty"`
}

func (i *Store) GetObservedGeneration() int64                { return i.Status.ObservedGeneration }
func (i *Store) SetObservedGeneration(g int64)               { i.Status.ObservedGeneration = g }
func (i *Store) GetNextReconcileTime() metav1.Time           { return i.Status.NextReconcileTime }
func (i *Store) SetNextReconcileTime(time metav1.Time)       { i.Status.NextReconcileTime = time }
func (i *Store) GetConditions() []metav1.Condition           { return i.Status.Conditions }
func (i *Store) SetConditions(conditions []metav1.Condition) { i.Status.Conditions = conditions }

//+kubebuilder:object:root=true

// StoreList contains a list of Store
type StoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Store `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Store{}, &StoreList{})
}
