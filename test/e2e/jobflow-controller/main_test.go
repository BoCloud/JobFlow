package jobflow_controller

import (
	"os"
	"testing"

	"jobflow/test/e2e/util"
)

func TestMain(m *testing.M) {
	mgr := util.NewManager()
	util.JobFlowReconciler = util.NewJobFlowReconciler(mgr)
	util.JobTemplateReconciler = util.NewJobTemplateReconciler(mgr)
	util.StartMgr(mgr)
	util.InitKubeClient()
	os.Exit(m.Run())
}
