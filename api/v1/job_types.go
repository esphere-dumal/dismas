package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobSpec defines the desired state of Job
type JobSpec struct {
	// +kubebuilder:validation:Required
	Command string `json:"command"`
	// +kubebuilder:validation:Optional
	Args []string `json:"args"`
}

// JobStatus defines the observed state of Job
type JobStatus struct {
	// TODO: using a struct 
	// Stdouts record all outputs with key refers to pod, value refers to output
	Stdouts map[string]string `json:"stdouts,omitempty"`
	// Stderrs records all error messages outputs with key refers to pod, value refers to error messages
	Stderrs map[string]string `json:"stderrs,omitempty"`
	// Errors records error when execute command in containers
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
