package validate

import (
	"fmt"

	"k8s.io/api/admission/v1beta1"
	whv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog"
	k8score "k8s.io/kubernetes/pkg/apis/core"
	k8scorev1 "k8s.io/kubernetes/pkg/apis/core/v1"
	v1qos "k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	k8scorevalid "k8s.io/kubernetes/pkg/apis/core/validation"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
	"jobflow/utils"
	"jobflow/webhooks/router"
	"jobflow/webhooks/schema"
)

func init() {
	_ = router.RegisterAdmission(service)
}

var service = &router.AdmissionService{
	Path: "/jobtemplates/validate",
	Func: AdmitJobTemplates,

	ValidatingConfig: &whv1beta1.ValidatingWebhookConfiguration{
		Webhooks: []whv1beta1.ValidatingWebhook{{
			Name: "validatetemplate.volcano.sh",
			Rules: []whv1beta1.RuleWithOperations{
				{
					Operations: []whv1beta1.OperationType{whv1beta1.Create},
					Rule: whv1beta1.Rule{
						APIGroups:   []string{"flow.volcano.sh"},
						APIVersions: []string{"v1alpha1"},
						Resources:   []string{"jobtemplates"},
					},
				},
			},
		}},
	},
}

// AdmitJobFlows is to admit jobFlows and return response.
func AdmitJobTemplates(ar v1beta1.AdmissionReview) error {
	klog.V(3).Infof("admitting jobtemplates -- %s", ar.Request.Operation)

	jobTemplate, err := schema.DecodeJobTemplate(ar.Request.Object, ar.Request.Resource)
	if err != nil {
		return err
	}

	switch ar.Request.Operation {
	case v1beta1.Create:
		err = validateJobTemplateCreate(jobTemplate)
	default:
		err = fmt.Errorf("only support 'CREATE' operation")
	}

	return err
}

func validateJobTemplateCreate(job *jobflowv1alpha1.JobTemplate) error {
	klog.V(3).Infof("validate create %s", job.Name)
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
		if err := validateK8sPodNameLength(podName); err != nil {
			msg += err.Error()
		}
		if err := validateTaskTemplate(task, job, index); err != nil {
			msg += err.Error()
		}
	}

	if err := validateJobName(job); err != nil {
		msg += err.Error()
	}

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

func validateTaskTemplate(task v1alpha1.TaskSpec, job *jobflowv1alpha1.JobTemplate, index int) error {
	var v1PodTemplate v1.PodTemplate
	v1PodTemplate.Template = *task.Template.DeepCopy()
	k8scorev1.SetObjectDefaults_PodTemplate(&v1PodTemplate)

	var coreTemplateSpec k8score.PodTemplateSpec
	if err := k8scorev1.Convert_v1_PodTemplateSpec_To_core_PodTemplateSpec(&v1PodTemplate.Template, &coreTemplateSpec, nil); err != nil {
		return fmt.Errorf("failed to convert v1_PodTemplateSpec to core_PodTemplateSpec")
	}

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
		return fmt.Errorf(msg)
	}

	err := validateTaskTopoPolicy(task, index)
	if err != nil {
		return err
	}

	return nil
}

// MakePodName creates pod name.
func makePodName(jobName string, taskName string, index int) string {
	return fmt.Sprintf("%s-%s-%d", jobName, taskName, index)
}
func validateK8sPodNameLength(podName string) error {
	if errMsgs := validation.IsQualifiedName(podName); len(errMsgs) > 0 {
		return fmt.Errorf(" create pod with name %s validate failed %v", podName, errMsgs)
	}
	return nil
}
func validateJobName(job *jobflowv1alpha1.JobTemplate) error {
	if errMsgs := validation.IsQualifiedName(job.Name); len(errMsgs) > 0 {
		return fmt.Errorf(" create job with name %s validate failed %v", job.Name, errMsgs)
	}
	return nil
}

func validateTaskTopoPolicy(task v1alpha1.TaskSpec, index int) error {
	if task.TopologyPolicy == "" || task.TopologyPolicy == v1alpha1.None {
		return nil
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
		return fmt.Errorf("spec.task[%d] isn't Guaranteed pod, kind=%v", index, v1qos.GetPodQOS(pod))
	}

	for id, container := range append(template.Spec.Containers, template.Spec.InitContainers...) {
		requestNum := guaranteedCPUs(container)
		if requestNum == 0 {
			return fmt.Errorf("the cpu request isn't  an integer in spec.task[%d] container[%d].",
				index, id)
		}
	}

	return nil
}

func guaranteedCPUs(container v1.Container) int {
	cpuQuantity := container.Resources.Requests[v1.ResourceCPU]
	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return 0
	}

	return int(cpuQuantity.Value())
}
