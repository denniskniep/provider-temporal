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

// TemporalNamespaceParameters are the configurable fields of a TemporalNamespace.
type TemporalNamespaceParameters struct {
	Name string `json:"name"`

	Description string `json:"description"`

	OwnerEmail string `json:"ownerEmail"`
}

// TemporalNamespaceObservation are the observable fields of a TemporalNamespace.
type TemporalNamespaceObservation struct {
	Id string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	OwnerEmail string `json:"ownerEmail"`

	State string `json:"state"`
}

// A TemporalNamespaceSpec defines the desired state of a TemporalNamespace.
type TemporalNamespaceSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TemporalNamespaceParameters `json:"forProvider"`
}

// A TemporalNamespaceStatus represents the observed state of a TemporalNamespace.
type TemporalNamespaceStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TemporalNamespaceObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A TemporalNamespace is an API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,temporal}
type TemporalNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemporalNamespaceSpec   `json:"spec"`
	Status TemporalNamespaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TemporalNamespaceList contains a list of TemporalNamespace
type TemporalNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TemporalNamespace `json:"items"`
}

// TemporalNamespace type metadata.
var (
	TemporalNamespaceKind             = reflect.TypeOf(TemporalNamespace{}).Name()
	TemporalNamespaceGroupKind        = schema.GroupKind{Group: Group, Kind: TemporalNamespaceKind}.String()
	TemporalNamespaceKindAPIVersion   = TemporalNamespaceKind + "." + SchemeGroupVersion.String()
	TemporalNamespaceGroupVersionKind = SchemeGroupVersion.WithKind(TemporalNamespaceKind)
)

func init() {
	SchemeBuilder.Register(&TemporalNamespace{}, &TemporalNamespaceList{})
}
