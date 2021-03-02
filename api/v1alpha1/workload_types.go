/*


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

const (
	PhasePending = "PENDING"
	PhaseCreated = "CREATED"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkloadSpec defines the desired state of Workload
type WorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Host is the name of the service from the service registry
	Host string `json:"host,omitempty"`

	Targets []DeploymentTarget `json:"targets,omitempty"`
}

type DeploymentTarget struct {
	Name         string       `json:"name,omitempty"`
	Namespace    string       `json:"namespace,omitempty"`
	TrafficSplit TrafficSplit `json:"trafficSplit,omitempty"`
}

type TrafficSplit struct {
	Hosts           []string          `json:"hosts,omitempty"`
	SubsetName      string            `json:"subsetName,omitempty"`
	SubsetLabels    map[string]string `json:"subsetLabels,omitempty"`
	DestinationHost string            `json:"destinationHost,omitempty"`
	Weight          int               `json:"weight,omitempty"`
}

// WorkloadStatus defines the observed state of Workload
type WorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	VirtualServiceState  string `json:"virtualServiceState,omitempty"`
	DestinationRuleState string `json:"destinationRuleState,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Workload is the Schema for the workloads API
type Workload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadSpec   `json:"spec,omitempty"`
	Status WorkloadStatus `json:"status,omitempty"`
}

func (w *Workload) GetIstioResourceName() string {
	return w.Spec.Host // strings.Split(w.Spec.Targets[0].Name, "-")[0]
}

func (w *Workload) GetHosts() []string {
	// need hosts as a set data structure
	hostSet := make(map[string]bool)
	hosts := []string{}

	for _, target := range w.Spec.Targets {
		for _, host := range target.TrafficSplit.Hosts {
			hostSet[host] = true
		}
	}

	for host := range hostSet {
		hosts = append(hosts, host)
	}

	return hosts
}

// +kubebuilder:object:root=true

// WorkloadList contains a list of Workload
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workload{}, &WorkloadList{})
}
