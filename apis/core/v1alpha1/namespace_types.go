/*
Copyright 2022 The Crossplane Authors.

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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// NamespaceParameters are the configurable fields of a Namespace.
type NamespaceParameters struct {
	Name string `json:"name"`

	Description string `json:"description"`

	OwnerEmail string `json:"ownerEmail"`
}

// NamespaceObservation are the observable fields of a Namespace.
type NamespaceObservation struct {
	Id string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	OwnerEmail string `json:"ownerEmail"`
}

// A NamespaceSpec defines the desired state of a Namespace.
type NamespaceSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       NamespaceParameters `json:"forProvider"`
}

// A NamespaceStatus represents the observed state of a Namespace.
type NamespaceStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NamespaceObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Namespace is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,temporal}
type Namespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespaceSpec   `json:"spec"`
	Status NamespaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NamespaceList contains a list of Namespace
type NamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Namespace `json:"items"`
}

// Namespace type metadata.
var (
	NamespaceKind             = reflect.TypeOf(Namespace{}).Name()
	NamespaceGroupKind        = schema.GroupKind{Group: Group, Kind: NamespaceKind}.String()
	NamespaceKindAPIVersion   = NamespaceKind + "." + SchemeGroupVersion.String()
	NamespaceGroupVersionKind = SchemeGroupVersion.WithKind(NamespaceKind)
)

func init() {
	SchemeBuilder.Register(&Namespace{}, &NamespaceList{})
}
