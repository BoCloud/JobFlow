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
	"fmt"

	"jobflow/utils"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var jobFlowLog = logf.Log.WithName("jobFlow-resource")

func (r *JobFlow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-batch-volcano-sh-v1alpha1-jobflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=batch.volcano.sh,resources=jobflows,verbs=create;update,versions=v1alpha1,name=mjobflow.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &JobFlow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *JobFlow) Default() {
	jobFlowLog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-batch-volcano-sh-v1alpha1-jobflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.volcano.sh,resources=jobflows,versions=v1alpha1,name=vjobflow.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &JobFlow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *JobFlow) ValidateCreate() error {
	jobFlowLog.Info("validate create", "name", r.Name)
	flows := r.Spec.Flows
	var msg string
	templateNames := map[string][]string{}
	vertexMap := make(map[string]*utils.Vertex)
	dag := &utils.DAG{}
	var duplicatedTemplate = false
	for _, template := range flows {
		if _, found := templateNames[template.Name]; found {
			// duplicate task name
			msg += fmt.Sprintf(" duplicated template name %s;", template.Name)
			duplicatedTemplate = true
			break
		} else {
			templateNames[template.Name] = template.DependsOn.Targets
			vertexMap[template.Name] = &utils.Vertex{Key: template.Name}
		}
	}
	if !duplicatedTemplate {
		for current, parents := range templateNames {
			if parents != nil && len(parents) > 0 {
				for _, parent := range parents {
					if _, found := vertexMap[parent]; !found {
						return fmt.Errorf("cannot find the template: %s ", parent)
					}
					dag.AddEdge(vertexMap[parent], vertexMap[current])
				}
			}
		}
		for k := range vertexMap {
			if err := dag.BFS(vertexMap[k]); err != nil {
				msg += fmt.Sprintf("%v;", err)
				break
			}
		}
	}

	if msg != "" {
		return fmt.Errorf(msg)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *JobFlow) ValidateUpdate(old runtime.Object) error {
	jobFlowLog.Info("validate update", "name", r.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *JobFlow) ValidateDelete() error {
	jobFlowLog.Info("validate delete", "name", r.Name)
	return nil
}
