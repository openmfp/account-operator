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

type AccountRole string

const (
	AccountRoleFolder  AccountRole = "Folder"
	AccountRoleAccount AccountRole = "Account"
)

// AccountSpec defines the desired state of Account
type AccountSpec struct {
	// AccountRole specifies the intended role for this Account object.
	// +kubebuilder:validation:Enum=Folder;Account
	AccountRole AccountRole `json:"accountRole"`

	// Existing Namespace is the account should take ownership of
	ExistingNamespace *string `json:"existingNamespace,omitempty"`

	// The display name for this account
	// +kubebuilder:validation:MaxLength=255
	DisplayName string `json:"displayName"`

	// An optional description for this account
	Description string `json:"description,omitempty"`

	// The initial creator of this account
	Creator string `json:"creator"`
}

// AccountStatus defines the observed state of Account
type AccountStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	Namespace          *string            `json:"namespace,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`
	NextReconcileTime  metav1.Time        `json:"nextReconcileTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.displayName",name="Display Name",type=string
// +kubebuilder:printcolumn:JSONPath=".status.Namespace",name="Account Namespace",type=string
// +kubebuilder:printcolumn:JSONPath=".spec.accountRole",name="Account Role",type=string
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// Account is the Schema for the accounts API
type Account struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccountSpec   `json:"spec,omitempty"`
	Status AccountStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AccountList contains a list of Account
type AccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Account `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Account{}, &AccountList{})
}

func (i *Account) GetObservedGeneration() int64          { return i.Status.ObservedGeneration }
func (i *Account) SetObservedGeneration(g int64)         { i.Status.ObservedGeneration = g }
func (i *Account) GetNextReconcileTime() metav1.Time     { return i.Status.NextReconcileTime }
func (i *Account) SetNextReconcileTime(time metav1.Time) { i.Status.NextReconcileTime = time }
