/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobSpec defines the desired state of Job
type JobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Minimun=0
	Command string `json:"command"`
	// +kubebuilder:validation:Minimun=0
	Args []string `json:"args"`
}

// JobStatus defines the observed state of Job
type JobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	LatestOutput string `json:"latestOutput,omitempty"`
	LatestError  string `json:"latestError,omitempty"`
	// Outputs record all outputs with key refers to pod, value refers to output
	Outputs map[string]string `json:"outputs,omitempty"`
	// Errors record all outputs with key refers to pod, value refers to output
	Errors map[string]string `json:"errors,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Job is the Schema for the jobs API
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobSpec   `json:"spec,omitempty"`
	Status JobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JobList contains a list of Job
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Job{}, &JobList{})
}
