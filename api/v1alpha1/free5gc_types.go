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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ComponentSpec defines the common configuration for Free5GC components
type ComponentSpec struct {
	// Image is the container image to use for the component
	Image string `json:"image"`
	// Replicas is the number of replicas to deploy
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Resources specifies the compute resources for the component
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Config is the component-specific configuration
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// NetworkSpec defines the network configuration for Free5GC components
type NetworkSpec struct {
	// N2Network is the network configuration for N2 interface (NGAP)
	// +optional
	N2Network *NetworkAttachmentConfig `json:"n2Network,omitempty"`
	// N3Network is the network configuration for N3 interface (User Plane)
	// +optional
	N3Network *NetworkAttachmentConfig `json:"n3Network,omitempty"`
	// N4Network is the network configuration for N4 interface (PFCP)
	// +optional
	N4Network *NetworkAttachmentConfig `json:"n4Network,omitempty"`
	// N6Network is the network configuration for N6 interface (Internet)
	// +optional
	N6Network *NetworkAttachmentConfig `json:"n6Network,omitempty"`
	// N9Network is the network configuration for N9 interface (F1-U)
	// +optional
	N9Network *NetworkAttachmentConfig `json:"n9Network,omitempty"`
}

// NetworkAttachmentConfig defines the configuration for a network attachment
type NetworkAttachmentConfig struct {
	// Name is the name of the NetworkAttachmentDefinition
	Name string `json:"name"`
	// Interface is the name of the interface in the pod
	Interface string `json:"interface"`
	// Type is the type of network (ipvlan, macvlan)
	// +optional
	Type string `json:"type,omitempty"`
	// Mode is the mode for ipvlan (l2, l3)
	// +optional
	Mode string `json:"mode,omitempty"`
	// Master is the master interface for the network
	// +optional
	MasterInterface string `json:"masterInterface,omitempty"`
	// Subnet is the subnet IP address
	// +optional
	Subnet string `json:"subnet,omitempty"`
	// CIDR is the network CIDR
	// +optional
	CIDR string `json:"cidr,omitempty"`
	// Gateway is the gateway IP address
	// +optional
	Gateway string `json:"gateway,omitempty"`
	// ExcludeIP is the IP to exclude from allocation
	// +optional
	ExcludeIP string `json:"excludeIP,omitempty"`
	// StaticIP is the static IP address to assign
	// +optional
	StaticIP string `json:"staticIP,omitempty"`
}

// Free5GCSpec defines the desired state of Free5GC
type Free5GCSpec struct {
	// MongoDB configuration
	// +optional
	MongoDB *MongoDBSpec `json:"mongodb,omitempty"`

	// NRF (Network Repository Function) configuration
	// +optional
	NRF *ComponentSpec `json:"nrf,omitempty"`

	// AMF (Access and Mobility Management Function) configuration
	// +optional
	AMF *ComponentSpec `json:"amf,omitempty"`

	// SMF (Session Management Function) configuration
	// +optional
	SMF *ComponentSpec `json:"smf,omitempty"`

	// UPF (User Plane Function) configuration
	// +optional
	UPF *UPFSpec `json:"upf,omitempty"`

	// AUSF (Authentication Server Function) configuration
	// +optional
	AUSF *ComponentSpec `json:"ausf,omitempty"`

	// NSSF (Network Slice Selection Function) configuration
	// +optional
	NSSF *ComponentSpec `json:"nssf,omitempty"`

	// PCF (Policy Control Function) configuration
	// +optional
	PCF *ComponentSpec `json:"pcf,omitempty"`

	// UDM (Unified Data Management) configuration
	// +optional
	UDM *ComponentSpec `json:"udm,omitempty"`

	// UDR (Unified Data Repository) configuration
	// +optional
	UDR *ComponentSpec `json:"udr,omitempty"`

	// N3IWF (Non-3GPP InterWorking Function) configuration
	// +optional
	N3IWF *ComponentSpec `json:"n3iwf,omitempty"`

	// WebUI configuration
	// +optional
	WebUI *ComponentSpec `json:"webui,omitempty"`

	// Network configuration for the Free5GC deployment
	// +optional
	Network NetworkSpec `json:"network,omitempty"`
}

// MongoDBSpec defines the configuration for MongoDB
type MongoDBSpec struct {
	// Use external MongoDB instead of deploying one
	// +optional
	External bool `json:"external,omitempty"`
	// URI is the connection URI for external MongoDB
	// +optional
	URI string `json:"uri,omitempty"`
	// Image is the container image to use for MongoDB
	// +optional
	Image string `json:"image,omitempty"`
	// Storage configuration for MongoDB
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`
}

// StorageSpec defines the storage configuration
type StorageSpec struct {
	// Size is the size of the storage volume
	Size string `json:"size"`
	// StorageClassName is the name of the storage class to use
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// UPFSpec defines the configuration for UPF
type UPFSpec struct {
	ComponentSpec `json:",inline"`
	// ULCL (Uplink Classifier) configuration
	// +optional
	ULCL *ULCLSpec `json:"ulcl,omitempty"`
}

// ULCLSpec defines the configuration for ULCL
type ULCLSpec struct {
	// Enable ULCL feature
	Enabled bool `json:"enabled"`
	// UPF instances configuration when ULCL is enabled
	// +optional
	Instances []UPFInstance `json:"instances,omitempty"`
}

// UPFInstance defines the configuration for a UPF instance in ULCL mode
type UPFInstance struct {
	// Name of the UPF instance
	Name string `json:"name"`
	// Configuration for this UPF instance
	ComponentSpec `json:",inline"`
}

// Free5GCStatus defines the observed state of Free5GC
type Free5GCStatus struct {
	// Conditions represent the latest available observations of the Free5GC state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// MongoDB represents the status of the MongoDB deployment
	// +optional
	MongoDB ComponentStatus `json:"mongodb,omitempty"`

	// Components represents the status of each Free5GC component
	// +optional
	Components map[string]ComponentStatus `json:"components,omitempty"`
}

// ComponentStatus defines the observed state of a component
type ComponentStatus struct {
	// Phase is the current phase of the component
	Phase string `json:"phase"`
	// Message provides more detail about the Phase
	// +optional
	Message string `json:"message,omitempty"`
	// Ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// Total replicas
	// +optional
	Replicas int32 `json:"replicas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Free5GC is the Schema for the free5gcs API
type Free5GC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Free5GCSpec   `json:"spec,omitempty"`
	Status Free5GCStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Free5GCList contains a list of Free5GC
type Free5GCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Free5GC `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Free5GC{}, &Free5GCList{})
}
