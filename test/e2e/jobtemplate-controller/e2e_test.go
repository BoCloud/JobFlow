package jobtemplate_controller

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"jobflow/test/e2e/util"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Volcano Job Seq Test Suite")
	util.Cancel()
}
