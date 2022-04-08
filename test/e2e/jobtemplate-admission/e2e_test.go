package jobtemplate_admission

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	e2eutil "jobflow/test/e2e/util"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Volcano JobTemplate Admission Test Suite")
	e2eutil.Cancel()
}
