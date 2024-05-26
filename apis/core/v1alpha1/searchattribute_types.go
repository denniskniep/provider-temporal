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

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// SearchAttributeParameters are the configurable fields of a SearchAttribute.
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.temporalNamespaceName) || has(self.temporalNamespaceName)", message="TemporalNamespaceName is required once set"
type SearchAttributeParameters struct {

	// Name of the SearchAttribute (immutable)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Name is immutable"
	Name string `json:"name"`

	// Type of the SearchAttribute (immutable)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Type is immutable"
	// +kubebuilder:validation:Enum=Text;Keyword;Int;Double;Bool;Datetime;KeywordList;
	Type string `json:"type"`

	// Namespace where search-attribute will be created (immutable)
	// At least one of temporalNamespaceName, temporalNamespaceNameRef or temporalNamespaceNameSelector is required.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="TemporalNamespaceName is immutable"
	// +crossplane:generate:reference:type=github.com/denniskniep/provider-temporal/apis/core/v1alpha1.TemporalNamespace
	TemporalNamespaceName *string `json:"temporalNamespaceName,omitempty"`

	// Namespace reference to retrieve the namespace name, where search-attribute will be created
	// At least one of temporalNamespaceName, temporalNamespaceNameRef or temporalNamespaceNameSelector is required.
	// +optional
	TemporalNamespaceNameRef *xpv1.Reference `json:"temporalNamespaceNameRef,omitempty"`

	// TemporalNamespaceNameSelector selects a reference to a TemporalNamespace and retrieves its name
	// At least one of temporalNamespaceName, temporalNamespaceNameRef or temporalNamespaceNameSelector is required.
	// +optional
	TemporalNamespaceNameSelector *xpv1.Selector `json:"temporalNamespaceNameSelector,omitempty"`
}

// SearchAttributeObservation are the observable fields of a SearchAttribute.
type SearchAttributeObservation struct {
	Name string `json:"name"`

	Type string `json:"type"`

	TemporalNamespaceName string `json:"temporalNamespaceName"`
}

// A SearchAttributeSpec defines the desired state of a SearchAttribute.
type SearchAttributeSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	// +kubebuilder:default={"name": "default"}
	ProviderReference *v1.Reference             `json:"providerRef,omitempty"`
	ForProvider       SearchAttributeParameters `json:"forProvider"`
}

// A SearchAttributeStatus represents the observed state of a SearchAttribute.
type SearchAttributeStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          SearchAttributeObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A SearchAttribute is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,temporal}
type SearchAttribute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SearchAttributeSpec   `json:"spec"`
	Status SearchAttributeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SearchAttributeList contains a list of SearchAttribute
type SearchAttributeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SearchAttribute `json:"items"`
}

// SearchAttribute type metadata.
var (
	SearchAttributeKind             = reflect.TypeOf(SearchAttribute{}).Name()
	SearchAttributeGroupKind        = schema.GroupKind{Group: Group, Kind: SearchAttributeKind}.String()
	SearchAttributeKindAPIVersion   = SearchAttributeKind + "." + SchemeGroupVersion.String()
	SearchAttributeGroupVersionKind = SchemeGroupVersion.WithKind(SearchAttributeKind)
)

func init() {
	SchemeBuilder.Register(&SearchAttribute{}, &SearchAttributeList{})
}
