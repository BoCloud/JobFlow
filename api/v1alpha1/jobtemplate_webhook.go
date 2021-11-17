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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	k8score "k8s.io/kubernetes/pkg/apis/core"
	k8scorev1 "k8s.io/kubernetes/pkg/apis/core/v1"
	v1qos "k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	k8scorevalid "k8s.io/kubernetes/pkg/apis/core/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

// log is for logging in this package.
var jobTemplateLog = logf.Log.WithName("jobtemplate-resource")

func (job *JobTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(job).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-batch-volcano-sh-v1alpha1-jobtemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=batch.volcano.sh,resources=jobtemplates,verbs=create;update,versions=v1alpha1,name=mjobtemplate.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Defaulter = &JobTemplate{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (job *JobTemplate) Default() {
	jobTemplateLog.Info("default", "name", job.Name)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-batch-volcano-sh-v1alpha1-jobtemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.volcano.sh,resources=jobtemplates,versions=v1alpha1,name=vjobtemplate.kb.io,admissionReviewVersions={v1,v1alpha1}

var _ webhook.Validator = &JobTemplate{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (job *JobTemplate) ValidateCreate() error {
	jobTemplateLog.Info("validate create", "name", job.Name)

	var msg string
	taskNames := map[string]string{}
	var totalReplicas int32

	if job.Spec.MinAvailable < 0 {
		return fmt.Errorf("job 'minAvailable' must be >= 0")
	}

	if job.Spec.MaxRetry < 0 {
		return fmt.Errorf("'maxRetry' cannot be less than zero")
	}

	if job.Spec.TTLSecondsAfterFinished != nil && *job.Spec.TTLSecondsAfterFinished < 0 {
		return fmt.Errorf("'ttlSecondsAfterFinished' cannot be less than zero")
	}

	if len(job.Spec.Tasks) == 0 {
		return fmt.Errorf("no task specified in job spec")
	}

	for index, task := range job.Spec.Tasks {
		if task.Replicas < 0 {
			msg += fmt.Sprintf(" 'replicas' < 0 in task: %s;", task.Name)
		}

		if task.MinAvailable != nil && *task.MinAvailable > task.Replicas {
			msg += fmt.Sprintf(" 'minAvailable' is greater than 'replicas' in task: %s, job: %s", task.Name, job.Name)
		}

		// count replicas
		totalReplicas += task.Replicas

		// validate task name
		if errMsgs := validation.IsDNS1123Label(task.Name); len(errMsgs) > 0 {
			msg += fmt.Sprintf(" %v;", errMsgs)
		}

		// duplicate task name
		if _, found := taskNames[task.Name]; found {
			msg += fmt.Sprintf(" duplicated task name %s;", task.Name)
			break
		} else {
			taskNames[task.Name] = task.Name
		}

		podName := makePodName(job.Name, task.Name, index)
		msg += validateK8sPodNameLength(podName)
		msg += validateTaskTemplate(task, job, index)
	}

	msg += validateJobName(job)

	if totalReplicas < job.Spec.MinAvailable {
		msg += "job 'minAvailable' should not be greater than total replicas in tasks;"
	}

	if err := utils.ValidatePolicies(job.Spec.Policies, field.NewPath("spec.policies")); err != nil {
		msg = msg + err.Error() + fmt.Sprintf(" valid events are %v, valid actions are %v;",
			utils.GetValidEvents(), utils.GetValidActions())
	}

	if err := utils.ValidateIO(job.Spec.Volumes); err != nil {
		msg += err.Error()
	}

	if msg != "" {
		return fmt.Errorf(msg)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (job *JobTemplate) ValidateUpdate(old runtime.Object) error {
	jobTemplateLog.Info("validate update", "name", job.Name)

	return fmt.Errorf("JobTemplate does not support update operations")
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (job *JobTemplate) ValidateDelete() error {
	jobTemplateLog.Info("validate delete", "name", job.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func validateTaskTemplate(task v1alpha1.TaskSpec, job *JobTemplate, index int) string {
	var v1PodTemplate v1.PodTemplate
	v1PodTemplate.Template = *task.Template.DeepCopy()
	k8scorev1.SetObjectDefaults_PodTemplate(&v1PodTemplate)

	var coreTemplateSpec k8score.PodTemplateSpec
	k8scorev1.Convert_v1_PodTemplateSpec_To_core_PodTemplateSpec(&v1PodTemplate.Template, &coreTemplateSpec, nil)

	// Skip verify container SecurityContex.Privileged as it depends on
	// the kube-apiserver `allow-privileged` flag.
	for i, container := range coreTemplateSpec.Spec.Containers {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
			coreTemplateSpec.Spec.Containers[i].SecurityContext.Privileged = nil
		}
	}

	corePodTemplate := k8score.PodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.Name,
			Namespace: job.Namespace,
		},
		Template: coreTemplateSpec,
	}

	if allErrs := k8scorevalid.ValidatePodTemplate(&corePodTemplate); len(allErrs) > 0 {
		msg := fmt.Sprintf("spec.task[%d].", index)
		for index := range allErrs {
			msg += allErrs[index].Error() + ". "
		}
		return msg
	}

	msg := validateTaskTopoPolicy(task, index)
	if msg != "" {
		return msg
	}

	return ""
}

// MakePodName creates pod name.
func makePodName(jobName string, taskName string, index int) string {
	return fmt.Sprintf("%s-%s-%d", jobName, taskName, index)
}
func validateK8sPodNameLength(podName string) string {
	if errMsgs := validation.IsQualifiedName(podName); len(errMsgs) > 0 {
		return fmt.Sprintf("create pod with name %s validate failed %v;", podName, errMsgs)
	}
	return ""
}
func validateJobName(job *JobTemplate) string {
	if errMsgs := validation.IsQualifiedName(job.Name); len(errMsgs) > 0 {
		return fmt.Sprintf("create job with name %s validate failed %v", job.Name, errMsgs)
	}
	return ""
}

func validateTaskTopoPolicy(task v1alpha1.TaskSpec, index int) string {
	if task.TopologyPolicy == "" || task.TopologyPolicy == v1alpha1.None {
		return ""
	}

	template := task.Template.DeepCopy()

	for id, container := range template.Spec.Containers {
		if len(container.Resources.Requests) == 0 {
			template.Spec.Containers[id].Resources.Requests = container.Resources.Limits.DeepCopy()
		}
	}

	for id, container := range template.Spec.InitContainers {
		if len(container.Resources.Requests) == 0 {
			template.Spec.InitContainers[id].Resources.Requests = container.Resources.Limits.DeepCopy()
		}
	}

	pod := &v1.Pod{
		Spec: template.Spec,
	}

	if v1qos.GetPodQOS(pod) != v1.PodQOSGuaranteed {
		return fmt.Sprintf("spec.task[%d] isn't Guaranteed pod, kind=%v", index, v1qos.GetPodQOS(pod))
	}

	for id, container := range append(template.Spec.Containers, template.Spec.InitContainers...) {
		requestNum := guaranteedCPUs(container)
		if requestNum == 0 {
			return fmt.Sprintf("the cpu request isn't  an integer in spec.task[%d] container[%d].",
				index, id)
		}
	}

	return ""
}

func guaranteedCPUs(container v1.Container) int {
	cpuQuantity := container.Resources.Requests[v1.ResourceCPU]
	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return 0
	}

	return int(cpuQuantity.Value())
}
