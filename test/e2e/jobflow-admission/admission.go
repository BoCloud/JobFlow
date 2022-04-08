package jobflow_admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
	e2eutil "jobflow/test/e2e/util"
)

var _ = Describe("JobFlow E2E Test: Test Admission service", func() {

	It("jobFlow validate check: duplicate job name check when create", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobFlow := &jobflowv1alpha1.JobFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-jobflow",
				Namespace: ctx.Namespace,
			},
			Spec: jobflowv1alpha1.JobFlowSpec{
				Flows: []jobflowv1alpha1.Flow{
					{
						Name: "job-a",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{},
						},
					},
					{
						Name: "job-b",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{},
						},
					},
					{
						Name: "job-b",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{},
						},
					},
				},
				JobRetainPolicy: jobflowv1alpha1.Retain,
			},
		}

		_, err := e2eutil.CreateJobFlowInner(ctx, jobFlow)
		Expect(err).To(MatchError(ContainSubstring(`duplicated template name job-b`)))
	})

	It("jobFlow validate check: targets name check when create", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobFlow := &jobflowv1alpha1.JobFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-jobflow",
				Namespace: ctx.Namespace,
			},
			Spec: jobflowv1alpha1.JobFlowSpec{
				Flows: []jobflowv1alpha1.Flow{
					{
						Name: "job-a",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{},
						},
					},
					{
						Name: "job-b",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{"unKnow-job"},
						},
					},
				},
				JobRetainPolicy: jobflowv1alpha1.Retain,
			},
		}

		_, err := e2eutil.CreateJobFlowInner(ctx, jobFlow)
		Expect(err).To(MatchError(ContainSubstring(`cannot find the template: unKnow-job`)))
	})

	It("jobFlow validate check: dependencies check when create", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobFlow := &jobflowv1alpha1.JobFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-jobflow",
				Namespace: ctx.Namespace,
			},
			Spec: jobflowv1alpha1.JobFlowSpec{
				Flows: []jobflowv1alpha1.Flow{
					{
						Name: "job-a",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{"job-b", "job-c"},
						},
					},
					{
						Name: "job-b",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{"job-a", "job-c"},
						},
					},
					{
						Name: "job-c",
						DependsOn: &jobflowv1alpha1.DependsOn{
							Targets: []string{"job-a", "job-b"},
						},
					},
				},
				JobRetainPolicy: jobflowv1alpha1.Retain,
			},
		}

		_, err := e2eutil.CreateJobFlowInner(ctx, jobFlow)
		Expect(err).To(MatchError(ContainSubstring(`find bad dependency`)))
	})

})
