package jobtemplate_controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"jobflow/test/e2e/util"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("JobTemplate E2E Test", func() {
	It("will create success ", func() {
		context := util.InitTestContext(util.Options{})
		defer util.CleanupTestContext(context)

		jobTemplate := util.GetJobTemplateInstance("jobtemplatetest")

		jobTemplateRes := util.CreateJobTemplate(context, jobTemplate)
		err := wait.Poll(100*time.Millisecond, util.OneMinute, util.JobTemplateExist(context, jobTemplateRes))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobTemplate created")
	})
	It("will delete success", func() {
		context := util.InitTestContext(util.Options{})
		defer util.CleanupTestContext(context)

		jobTemplate := util.GetJobTemplateInstance("jobtemplatetest")

		util.CreateJobTemplate(context, jobTemplate)

		jobTemplateRes := util.DeleteJobTemplate(context, jobTemplate)
		err := wait.Poll(100*time.Millisecond, util.OneMinute, util.JobTemplateNotExist(context, jobTemplateRes))
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobTemplate deleted")
	})
	It("will update status success", func() {
		context := util.InitTestContext(util.Options{})
		defer util.CleanupTestContext(context)

		jobTemplate := util.GetJobTemplateInstance("jobtemplatetest")

		jobTemplateRes := util.CreateJobTemplate(context, jobTemplate)

		jobTemplateRes.Status.JobDependsOnList = []string{"jobflowtest"}

		jobTemplateUpdateRes := util.UpdateJobTemplateStatus(context, jobTemplateRes)
		err := wait.Poll(100*time.Millisecond, util.OneMinute, func() (done bool, err error) {
			if jobTemplateUpdateRes.Status.JobDependsOnList != nil && len(jobTemplateUpdateRes.Status.JobDependsOnList) > 0 {
				return true, nil
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred(), "failed to wait for JobTemplateStatus updated")
	})
})
