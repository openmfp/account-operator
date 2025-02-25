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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AccountInfoSpec defines the desired state of Account
type AccountInfoSpec struct {
	FGA           FGAInfo          `json:"fga"`
	Account       AccountLocation  `json:"account"`
	ParentAccount *AccountLocation `json:"parentAccount,omitempty"`
	Organization  AccountLocation  `json:"organization"`
	ClusterInfo   ClusterInfo      `json:"clusterInfo"`
}

type ClusterInfo struct {
	CA string `json:"ca"`
}

type AccountLocation struct {
	Name      string      `json:"name"`
	ClusterId string      `json:"clusterId"`
	Path      string      `json:"path"`
	URL       string      `json:"url"`
	Type      AccountType `json:"type"`
}

type FGAInfo struct {
	Store StoreInfo `json:"store"`
}

type StoreInfo struct {
	Id string `json:"id"`
}

// AccountInfoStatus defines the observed state of Account
type AccountInfoStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=accountinfos
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// Account is the Schema for the accounts API
type AccountInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccountInfoSpec   `json:"spec,omitempty"`
	Status AccountInfoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AccountInfoList contains a list of Account
type AccountInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Account `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AccountInfo{}, &AccountInfoList{})
}
