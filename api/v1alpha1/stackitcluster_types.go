/*
Copyright 2026.

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
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// ClusterFinalizer allows StackitClusterReconciler to clean up resources before
// the StackitCluster is finally removed from the API server.
const ClusterFinalizer = "stackitcluster.infrastructure.cluster.x-k8s.io"

// StackitClusterSpec defines the desired state of StackitCluster.
// +kubebuilder:validation:XValidation:rule="has(self.credentialsSecretRef.name) && size(self.credentialsSecretRef.name) > 0",message="credentialsSecretRef.name is required"
// +kubebuilder:validation:XValidation:rule="has(self.apiServerLoadBalancer) && has(self.apiServerLoadBalancer.enabled) && self.apiServerLoadBalancer.enabled ? true : has(self.controlPlaneEndpoint) && has(self.controlPlaneEndpoint.host) && size(self.controlPlaneEndpoint.host) > 0",message="controlPlaneEndpoint.host is required when apiServerLoadBalancer.enabled is false"
type StackitClusterSpec struct {
	// projectID is the STACKIT project that owns the cluster infrastructure.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F-]{36}$`
	ProjectID string `json:"projectID"`

	// region is the STACKIT region in which infrastructure is created.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-z]{2}[0-9]{2}$`
	Region string `json:"region"`

	// credentialsSecretRef references the Secret containing STACKIT credentials.
	// +required
	CredentialsSecretRef corev1.SecretReference `json:"credentialsSecretRef"`

	// network references the existing STACKIT network the cluster will use.
	// For MVP the network must already exist.
	// +required
	Network StackitClusterNetworkSpec `json:"network"`

	// apiServerLoadBalancer configures the load balancer in front of the
	// Kubernetes API server.
	// +optional
	APIServerLoadBalancer StackitAPIServerLoadBalancerSpec `json:"apiServerLoadBalancer,omitempty"`

	// controlPlaneEndpoint allows the user to provide a pre-existing endpoint
	// when apiServerLoadBalancer.enabled is false.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty,omitzero"`

	// additionalLabels is merged into the labels applied to cluster-wide
	// STACKIT resources such as the API server load balancer.
	// +optional
	// +kubebuilder:validation:MaxProperties=64
	AdditionalLabels map[string]string `json:"additionalLabels,omitempty"`
}

// StackitClusterNetworkSpec references an existing STACKIT network.
type StackitClusterNetworkSpec struct {
	// id is the STACKIT network ID.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F-]{36}$`
	ID string `json:"id"`
}

// StackitAPIServerLoadBalancerSpec configures an optional load balancer for
// the Kubernetes API server.
type StackitAPIServerLoadBalancerSpec struct {
	// enabled toggles creation of an API server load balancer.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// StackitClusterStatus defines the observed state of StackitCluster.
type StackitClusterStatus struct {
	// ready indicates that the infrastructure required for the cluster is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// initialization provides Cluster API v1beta2 contract state.
	// +optional
	Initialization StackitClusterInitializationStatus `json:"initialization,omitempty"`

	// apiServerEndpoint is the endpoint the Kubernetes API server is reachable at.
	// +optional
	APIServerEndpoint clusterv1.APIEndpoint `json:"apiServerEndpoint,omitempty,omitzero"`

	// apiServerLoadBalancerID stores the ID of a provider-managed API server
	// load balancer so it can be deleted on cluster teardown.
	// +optional
	APIServerLoadBalancerID string `json:"apiServerLoadBalancerID,omitempty"`

	// conditions represent the current state of the StackitCluster resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// StackitClusterInitializationStatus holds Cluster API initialization state.
type StackitClusterInitializationStatus struct {
	// provisioned is true when the cluster infrastructure is provisioned.
	// +optional
	Provisioned bool `json:"provisioned,omitempty"`
}

// Condition types maintained by the StackitClusterReconciler.
const (
	ClusterReadyCondition             = "Ready"
	ClusterNetworkReadyCondition      = "NetworkReady"
	ClusterLoadBalancerReadyCondition = "LoadBalancerReady"
	ClusterCredentialsReadyCondition  = "CredentialsReady"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=stackitclusters,shortName=stic,scope=Namespaced,categories=cluster-api
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// StackitCluster is the Schema for the stackitclusters API.
type StackitCluster struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of StackitCluster
	// +required
	Spec StackitClusterSpec `json:"spec"`

	// status defines the observed state of StackitCluster
	// +optional
	Status StackitClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// StackitClusterList contains a list of StackitCluster.
type StackitClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []StackitCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StackitCluster{}, &StackitClusterList{})
}
