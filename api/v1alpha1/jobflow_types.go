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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobFlowSpec defines the desired state of JobFlow
type JobFlowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Flow defines the dependent of jobs
	Flow []Flow `json:"flow"`
}

// Flow defines the dependent of jobs
type Flow struct {
	Name     string   `json:"name"`
	DependOn DependOn `json:"dependOn"`
}

type DependOn struct {
	Target []string `json:"target"`
}

// JobFlowStatus defines the observed state of JobFlow
type JobFlowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	PendingJobs    []string             `json:"pendingJobs"`
	RunningJobs    []string             `json:"runningJobs"`
	FailedJobs     []string             `json:"failedJobs"`
	CompletedJobs  []string             `json:"completedJobs"`
	TerminatedJobs []string             `json:"terminatedJobs"`
	UnKnowJobs     []string             `json:"unKnowJobs"`
	JobList        []string             `json:"jobList"`
	Conditions     map[string]Condition `json:"conditions"`
}

type Condition struct {
	Phase          JobPhase        `json:"phase"`
	CreateTime     *metav1.Time    `json:"createTime"`
	CompleteTime   *metav1.Time    `json:"completeTime"`
	TaskConditions []TaskCondition `json:"taskConditions"`
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

type TaskCondition struct {
	CompletedTasks []string `json:"completedTasks"`
	RunningTasks   []string `json:"runningTasks"`
}

// +kubebuilder:object:root=true

// JobFlow is the Schema for the jobflows API
type JobFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobFlowSpec   `json:"spec,omitempty"`
	Status JobFlowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JobFlowList contains a list of JobFlow
type JobFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobFlow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JobFlow{}, &JobFlowList{})
}
