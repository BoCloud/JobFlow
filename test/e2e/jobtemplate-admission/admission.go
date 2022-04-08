package jobtemplate_admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	busv1alpha1 "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
	e2eutil "jobflow/test/e2e/util"
)

var _ = Describe("JobTemplate E2E Test: Test Admission service", func() {

	It("jobTemplate validate check: duplicate task name check when create", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobTemplate := &jobflowv1alpha1.JobTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job-template",
				Namespace: ctx.Namespace,
			},
			Spec: v1alpha1.JobSpec{
				SchedulerName: "volcano",
				MinAvailable:  1,
				Tasks: []v1alpha1.TaskSpec{
					{
						Name:     "task1",
						Replicas: 1,
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nginx",
										Image: "nginx:1.14.2",
									},
								},
							},
						},
					},
					{
						Name:     "task1",
						Replicas: 1,
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nginx",
										Image: "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
			},
		}

		_, err := e2eutil.CreateJobTemplateInner(ctx, jobTemplate)
		Expect(err).To(MatchError(ContainSubstring(`duplicated task name task1`)))
	})

	It("jobTemplate validate check: minAvailable larger than replicas when create", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobTemplate := &jobflowv1alpha1.JobTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job-template",
				Namespace: ctx.Namespace,
			},
			Spec: v1alpha1.JobSpec{
				SchedulerName: "volcano",
				MinAvailable:  2,
				Tasks: []v1alpha1.TaskSpec{
					{
						Name:     "task1",
						Replicas: 1,
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nginx",
										Image: "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
			},
		}

		_, err := e2eutil.CreateJobTemplateInner(ctx, jobTemplate)
		Expect(err).To(MatchError(ContainSubstring(`job 'minAvailable' should not be greater than total replicas in tasks`)))
	})

	It("jobTemplate validate check: no task specified in the jobTemplate when create", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobTemplate := &jobflowv1alpha1.JobTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job-template",
				Namespace: ctx.Namespace,
			},
			Spec: v1alpha1.JobSpec{
				SchedulerName: "volcano",
				MinAvailable:  1,
				Tasks:         []v1alpha1.TaskSpec{},
			},
		}

		_, err := e2eutil.CreateJobTemplateInner(ctx, jobTemplate)
		Expect(err).To(MatchError(ContainSubstring(`no task specified in job spec`)))
	})

	It("jobTemplate validate check: invalid policy action when create", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobTemplate := &jobflowv1alpha1.JobTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job-template",
				Namespace: ctx.Namespace,
			},
			Spec: v1alpha1.JobSpec{
				SchedulerName: "volcano",
				Tasks: []v1alpha1.TaskSpec{
					{
						Name:     "task1",
						Replicas: 1,
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "nginx",
										Image: "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
				Policies: []v1alpha1.LifecyclePolicy{
					{
						Event:  busv1alpha1.PodEvictedEvent,
						Action: busv1alpha1.Action("someFakeAction"),
					},
				},
			},
		}

		_, err := e2eutil.CreateJobTemplateInner(ctx, jobTemplate)
		Expect(err).To(MatchError(ContainSubstring(`invalid policy action`)))
	})

})
