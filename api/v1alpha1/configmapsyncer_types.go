/*
Copyright 2025.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConfigMapSyncerSpec defines the desired state of ConfigMapSyncer.
type ConfigMapSyncerSpec struct {
	// MasterConfigMap is the reference to the source ConfigMap that will be propagated
	// +kubebuilder:validation:Required
	MasterConfigMap ConfigMapReference `json:"masterConfigMap"`

	// TargetConfigMapName is the name to use for target ConfigMaps
	// If not specified, the name of the master ConfigMap will be used
	// +optional
	TargetConfigMapName string `json:"targetConfigMapName,omitempty"`

	// TargetNamespaces is a list of namespaces where the ConfigMap should be propagated
	// +optional
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`

	// TargetSelector is a label selector to identify target ConfigMaps
	// +optional
	TargetSelector *metav1.LabelSelector `json:"targetSelector,omitempty"`

	// MergeStrategy defines how to handle conflicts when merging ConfigMaps
	// +kubebuilder:validation:Enum=Replace;Merge
	// +kubebuilder:default=Merge
	MergeStrategy string `json:"mergeStrategy,omitempty"`

	// SyncInterval is the interval between sync operations in seconds
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=300
	SyncInterval int32 `json:"syncInterval,omitempty"`
}

// ConfigMapReference contains information to reference a ConfigMap
type ConfigMapReference struct {
	// Name of the ConfigMap
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the ConfigMap
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
}

// SyncStatus represents the status of a ConfigMap sync operation
type SyncStatus struct {
	// ConfigMapName is the name of the target ConfigMap
	ConfigMapName string `json:"configMapName"`

	// Namespace is the namespace of the target ConfigMap
	Namespace string `json:"namespace"`

	// LastSyncTime is the timestamp of the last successful sync
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Status of the sync operation
	// +kubebuilder:validation:Enum=Pending;Synced;Failed
	Status string `json:"status"`

	// Message provides additional information about the sync status
	// +optional
	Message string `json:"message,omitempty"`
}

// ConfigMapSyncerStatus defines the observed state of ConfigMapSyncer.
type ConfigMapSyncerStatus struct {
	// Conditions represent the latest available observations of the ConfigMapSyncer's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// SyncStatuses contains the sync status for each target ConfigMap
	// +optional
	SyncStatuses []SyncStatus `json:"syncStatuses,omitempty"`

	// LastSyncTime is the timestamp of the last sync attempt
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ConfigMapSyncer is the Schema for the configmapsyncers API.
// +kubebuilder:resource:scope=Namespaced,shortName=cms
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ConfigMapSyncer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigMapSyncerSpec   `json:"spec,omitempty"`
	Status ConfigMapSyncerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConfigMapSyncerList contains a list of ConfigMapSyncer.
type ConfigMapSyncerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigMapSyncer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigMapSyncer{}, &ConfigMapSyncerList{})
}
