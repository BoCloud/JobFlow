package jobflow_admission

import (
	"os"
	"testing"

	e2eutil "jobflow/test/e2e/util"
)

func TestMain(m *testing.M) {
	mgr := e2eutil.NewManager()
	e2eutil.JobFlowReconciler = e2eutil.NewJobFlowReconciler(mgr)
	e2eutil.JobTemplateReconciler = e2eutil.NewJobTemplateReconciler(mgr)
	e2eutil.StartMgr(mgr)
	e2eutil.InitKubeClient()
	os.Exit(m.Run())
}
