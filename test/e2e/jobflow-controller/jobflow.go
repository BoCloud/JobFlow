package jobflow_controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"jobflow/test/e2e/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

var _ = Describe("JobFlow E2E Test", func() {
	It("will create success and deploy by flow", func() {
		ctx := util.InitTestContext(util.Options{})
		defer util.CleanupTestContext(ctx)

		//create jobtemplateA and jobtemplateB
		jobTemplateA := util.GetJobTemplateInstance("jobtemplate-a")
		jobTemplateB := util.GetJobTemplateInstance("jobtemplate-b")
		util.CreateJobTemplate(ctx, jobTemplateA)
		util.CreateJobTemplate(ctx, jobTemplateB)

		jobflow := util.GetFlowInstance("jobflowtest")

		jobFlowRes := util.CreateJobFlow(ctx, jobflow)
		err := wait.Poll(100*time.Millisecond, util.OneMinute, util.JobFlowExist(ctx, jobFlowRes))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobFlow created")

		err = wait.Poll(100*time.Millisecond, util.FiveMinute, util.VcJobExist(ctx, jobFlowRes, jobTemplateA))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for vcjob created")

		err = wait.Poll(100*time.Millisecond, util.FiveMinute, func() (done bool, err error) {
			jobB := types.NamespacedName{
				Namespace: jobTemplateB.Namespace,
				Name:      util.GetJobName(jobflow.Name, jobTemplateB.Name),
			}
			jobBGet := &v1alpha1.Job{}
			err = ctx.JobFlowReconciler.Get(context.TODO(), jobB, jobBGet)
			if err != nil {
				if errors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			jobA := types.NamespacedName{
				Namespace: jobTemplateA.Namespace,
				Name:      util.GetJobName(jobflow.Name, jobTemplateA.Name),
			}
			jobAGet := &v1alpha1.Job{}
			err = ctx.JobFlowReconciler.Get(context.TODO(), jobA, jobAGet)
			if err != nil {
				if errors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			if jobAGet.Status.State.Phase == v1alpha1.Completed && jobBGet.Status.State.Phase != v1alpha1.Completed {
				return true, nil
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobFlow success running")
	})
	It("will delete success ", func() {
		ctx := util.InitTestContext(util.Options{})
		defer util.CleanupTestContext(ctx)

		jobFlow := util.GetFlowInstance("jobflowtest")

		util.CreateJobFlow(ctx, jobFlow)

		jobFlowRes := util.DeleteJobFlow(ctx, jobFlow)
		err := wait.Poll(100*time.Millisecond, util.OneMinute, util.JobFlowNotExist(ctx, jobFlowRes))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobFlow created")
	})
	It("will update status success ", func() {
		ctx := util.InitTestContext(util.Options{})
		defer util.CleanupTestContext(ctx)

		jobFlow := util.GetFlowInstance("jobflowtest")

		jobFlowRes := util.CreateJobFlow(ctx, jobFlow)

		jobFlowRes.Status.PendingJobs = []string{"jobA"}

		jobFlowUpdateRes := util.UpdateJobFlowStatus(ctx, jobFlow)
		err := wait.Poll(100*time.Millisecond, util.OneMinute, func() (done bool, err error) {
			if jobFlowUpdateRes.Status.PendingJobs != nil && len(jobFlowUpdateRes.Status.PendingJobs) > 0 {
				return true, nil
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobFlowStatus updated")
	})
	It("will delete all completed vcJobs success ", func() {
		ctx := util.InitTestContext(util.Options{})
		defer util.CleanupTestContext(ctx)

		//create jobtemplateA and jobtemplateB
		jobTemplateA := util.GetJobTemplateInstance("jobtemplate-a")
		jobTemplateB := util.GetJobTemplateInstance("jobtemplate-b")
		util.CreateJobTemplate(ctx, jobTemplateA)
		util.CreateJobTemplate(ctx, jobTemplateB)

		jobflow := util.GetFlowInstanceRetainPolicyDelete("jobflowtest")

		jobFlowRes := util.CreateJobFlow(ctx, jobflow)
		err := wait.Poll(100*time.Millisecond, util.OneMinute, util.JobFlowExist(ctx, jobFlowRes))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobFlow created")

		err = wait.Poll(100*time.Millisecond, util.FiveMinute, util.VcJobExist(ctx, jobFlowRes, jobTemplateA))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for vcjob created")

		err = wait.Poll(100*time.Millisecond, util.FiveMinute, util.VcJobExist(ctx, jobFlowRes, jobTemplateB))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for vcjob created")

		err = wait.Poll(100*time.Millisecond, util.FiveMinute, util.VcJobNotExist(ctx, jobFlowRes, jobTemplateA))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for vcjob deleted")

		err = wait.Poll(100*time.Millisecond, util.FiveMinute, util.VcJobNotExist(ctx, jobFlowRes, jobTemplateB))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for vcjob deletde")
	})
})
