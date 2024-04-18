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

// ProcessParameters are the configurable fields of a Process.
type ProcessParameters struct {
	ConfigurableField string `json:"configurableField"`
}

// ProcessObservation are the observable fields of a Process.
type ProcessObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A ProcessSpec defines the desired state of a Process.
type ProcessSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ProcessParameters `json:"forProvider"`
}

// A ProcessStatus represents the observed state of a Process.
type ProcessStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ProcessObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Process is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,processprovider}
type Process struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProcessSpec   `json:"spec"`
	Status ProcessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProcessList contains a list of Process
type ProcessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Process `json:"items"`
}

// Process type metadata.
var (
	ProcessKind             = reflect.TypeOf(Process{}).Name()
	ProcessGroupKind        = schema.GroupKind{Group: Group, Kind: ProcessKind}.String()
	ProcessKindAPIVersion   = ProcessKind + "." + SchemeGroupVersion.String()
	ProcessGroupVersionKind = SchemeGroupVersion.WithKind(ProcessKind)
)

func init() {
	SchemeBuilder.Register(&Process{}, &ProcessList{})
}
