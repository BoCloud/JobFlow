/*
Copyright 2021.

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

// JobFlowSpec defines the desired state of JobFlow
type JobFlowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of JobFlow. Edit jobflow_types.go to remove/update
	Flow []Flow `json:"flow,omitempty"`
}

// Flow defines the dependent of jobs
type Flow struct {
	Name      string     `json:"name"`
	DependsOn *DependsOn `json:"dependsOn,omitempty"`
}

type DependsOn struct {
	Target []string `json:"target,omitempty"`
}

// JobFlowStatus defines the observed state of JobFlow
type JobFlowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	PendingJobs    []string             `json:"pendingJobs,omitempty"`
	RunningJobs    []string             `json:"runningJobs,omitempty"`
	FailedJobs     []string             `json:"failedJobs,omitempty"`
	CompletedJobs  []string             `json:"completedJobs,omitempty"`
	TerminatedJobs []string             `json:"terminatedJobs,omitempty"`
	UnKnowJobs     []string             `json:"unKnowJobs,omitempty"`
	JobList        []string             `json:"jobList,omitempty"`
	Conditions     map[string]Condition `json:"conditions,omitempty"`
}

type Condition struct {
	Phase         *JobPhase      `json:"phase,omitempty"`
	CreateTime    *metav1.Time   `json:"createTime,omitempty"`
	CompleteTime  *metav1.Time   `json:"completeTime,omitempty"`
	JobConditions []JobCondition `json:"jobConditions,omitempty"`
}

type JobPhase string

const (
	Running     JobPhase = "Running"
	Pending     JobPhase = "Pending"
	Succeeded   JobPhase = "Succeeded"
	Terminating JobPhase = "Terminating"
	Terminated  JobPhase = "Terminated"
	Unknown     JobPhase = "Unknown"
	Failed      JobPhase = "Failed"
	Waiting     JobPhase = "Waiting"
)

type JobCondition struct {
	SucceededJobs []string `json:"succeededJobs,omitempty"`
	RunningJobs   []string `json:"runningJobs,omitempty"`
	PendingJobs   []string `json:"pendingJobs,omitempty"`
	FailedJobs    []string `json:"failedJobs,omitempty"`
	UnknownJobs   []string `json:"unknownJobs,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// JobFlow is the Schema for the jobflows API
type JobFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobFlowSpec   `json:"spec,omitempty"`
	Status JobFlowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JobFlowList contains a list of JobFlow
type JobFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobFlow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JobFlow{}, &JobFlowList{})
}
