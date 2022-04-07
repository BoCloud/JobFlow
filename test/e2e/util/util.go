package util

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	flowv1alpha1 "jobflow/api/v1alpha1"
	"jobflow/controllers"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	busv1alpha1 "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

var (
	FiveMinute = 5 * time.Minute
	OneMinute  = 1 * time.Minute
)

var KubeClient *kubernetes.Clientset

var JobFlowReconciler *controllers.JobFlowReconciler

var Cancel context.CancelFunc

func StartMgr(mgr ctrl.Manager) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := mgr.Start(ctx); err != nil {
			os.Exit(1)
		}
	}()
	Cancel = cancel
}

func NewJobFlowReconciler(mgr ctrl.Manager) *controllers.JobFlowReconciler {
	jobFlowReconciler := &controllers.JobFlowReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("containerset-controller"),
	}
	return jobFlowReconciler
}

var JobTemplateReconciler *controllers.JobTemplateReconciler

func NewJobTemplateReconciler(mgr ctrl.Manager) *controllers.JobTemplateReconciler {
	jobTemplateReconciler := &controllers.JobTemplateReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("containerset-controller"),
	}
	return jobTemplateReconciler
}

func InitKubeClient() {
	home := HomeDir()
	configPath := KubeconfigPath(home)
	config, _ := clientcmd.BuildConfigFromFlags(MasterURL(), configPath)
	KubeClient = kubernetes.NewForConfigOrDie(config)
}

func NewManager() ctrl.Manager {
	scheme := runtime.NewScheme()
	setupLog := ctrl.Log.WithName("setup")

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(flowv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     ":8080",
		Port:                   9443,
		HealthProbeBindAddress: ":8081",
		LeaderElection:         false,
		LeaderElectionID:       "1b1c5f74.volcano.sh",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	return mgr
}

type Options struct {
	Namespace string
}

type TestContext struct {
	Namespace             string
	JobFlowReconciler     *controllers.JobFlowReconciler
	JobTemplateReconciler *controllers.JobTemplateReconciler
	KubeClient            *kubernetes.Clientset
}

func InitTestContext(o Options) *TestContext {
	By("Initializing test context")

	if o.Namespace == "" {
		o.Namespace = GenRandomStr(8)
	}
	ctx := &TestContext{
		Namespace:             o.Namespace,
		JobFlowReconciler:     JobFlowReconciler,
		JobTemplateReconciler: JobTemplateReconciler,
		KubeClient:            KubeClient,
	}

	_, err := ctx.KubeClient.CoreV1().Namespaces().Create(context.TODO(),
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ctx.Namespace,
			},
		},
		metav1.CreateOptions{},
	)
	Expect(err).NotTo(HaveOccurred(), "failed to create namespace")

	return ctx
}

func CleanupTestContext(ctx *TestContext) {
	By("Cleaning up test context")

	foreground := metav1.DeletePropagationForeground
	err := ctx.KubeClient.CoreV1().Namespaces().Delete(context.TODO(), ctx.Namespace, metav1.DeleteOptions{
		PropagationPolicy: &foreground,
	})
	Expect(err).NotTo(HaveOccurred(), "failed to delete namespace")

	// Wait for namespace deleted.
	err = wait.Poll(100*time.Millisecond, FiveMinute, NamespaceNotExist(ctx))
	Expect(err).NotTo(HaveOccurred(), "failed to wait for namespace deleted")
}

func NamespaceNotExist(ctx *TestContext) wait.ConditionFunc {
	return NamespaceNotExistWithName(ctx, ctx.Namespace)
}

func NamespaceNotExistWithName(ctx *TestContext, name string) wait.ConditionFunc {
	return func() (bool, error) {
		_, err := ctx.KubeClient.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	}
}

func GenRandomStr(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func MasterURL() string {
	if m := os.Getenv("MASTER"); m != "" {
		return m
	}
	return ""
}

func KubeconfigPath(home string) string {
	if m := os.Getenv("KUBECONFIG"); m != "" {
		return m
	}
	return filepath.Join(home, ".kube", "config") // default kubeconfig path is $HOME/.kube/config
}

func CreateJobTemplate(context *TestContext, jobTemplate *flowv1alpha1.JobTemplate) *flowv1alpha1.JobTemplate {
	jobTemplateRes, err := CreateJobTemplateInner(context, jobTemplate)
	Expect(err).NotTo(HaveOccurred(), "failed to create jobTemplate %s in namespace %s", jobTemplate.Name, jobTemplate.Namespace)
	return jobTemplateRes
}

func CreateJobTemplateInner(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) (*flowv1alpha1.JobTemplate, error) {
	jobTemplate.Namespace = ctx.Namespace
	err := ctx.JobTemplateReconciler.Create(context.TODO(), jobTemplate)
	return jobTemplate, err
}

func DeleteJobTemplate(context *TestContext, jobTemplate *flowv1alpha1.JobTemplate) *flowv1alpha1.JobTemplate {
	jobTemplateRes, err := DeleteJobTemplateInner(context, jobTemplate)
	Expect(err).NotTo(HaveOccurred(), "failed to delete jobTemplate %s in namespace %s", jobTemplate.Name, jobTemplate.Namespace)
	return jobTemplateRes
}

func DeleteJobTemplateInner(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) (*flowv1alpha1.JobTemplate, error) {
	jobTemplate.Namespace = ctx.Namespace
	err := ctx.JobTemplateReconciler.Delete(context.TODO(), jobTemplate)
	return jobTemplate, err
}

func UpdateJobTemplateStatus(context *TestContext, jobTemplate *flowv1alpha1.JobTemplate) *flowv1alpha1.JobTemplate {
	jobTemplateRes, err := UpdateJobTemplateStatusInner(context, jobTemplate)
	Expect(err).NotTo(HaveOccurred(), "failed to update jobTemplate %s in namespace %s", jobTemplate.Name, jobTemplate.Namespace)
	return jobTemplateRes
}

func UpdateJobTemplateStatusInner(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) (*flowv1alpha1.JobTemplate, error) {
	jobTemplate.Namespace = ctx.Namespace
	err := ctx.JobTemplateReconciler.Status().Update(context.TODO(), jobTemplate)
	return jobTemplate, err
}

func JobTemplateExist(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		key := types.NamespacedName{
			Namespace: jobTemplate.Namespace,
			Name:      jobTemplate.Name,
		}
		jobTemplateGet := &flowv1alpha1.JobTemplate{}
		err := ctx.JobTemplateReconciler.Get(context.TODO(), key, jobTemplateGet)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

func JobTemplateNotExist(ctx *TestContext, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		key := types.NamespacedName{
			Namespace: jobTemplate.Namespace,
			Name:      jobTemplate.Name,
		}
		jobTemplateGet := &flowv1alpha1.JobTemplate{}
		err := ctx.JobTemplateReconciler.Get(context.TODO(), key, jobTemplateGet)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}
}

func GetJobTemplateInstance(jobTemplateName string) *flowv1alpha1.JobTemplate {
	jobTemplate := &flowv1alpha1.JobTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobTemplateName,
		},
		Spec: v1alpha1.JobSpec{
			SchedulerName: "volcano",
			MinAvailable:  1,
			Volumes:       nil,
			Tasks: []v1alpha1.TaskSpec{
				{
					Name:         "tasktest",
					Replicas:     1,
					MinAvailable: nil,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: corev1.PodSpec{
							Volumes: nil,
							Containers: []corev1.Container{
								{
									Name:  "nginx",
									Image: "nginx:1.14.2",
									Command: []string{
										"sh",
										"-c",
										"sleep 10s",
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.TaskCompletedEvent,
							Action: busv1alpha1.CompleteJobAction,
						},
					},
					TopologyPolicy: "",
					MaxRetry:       0,
				},
			},
			Policies: []v1alpha1.LifecyclePolicy{
				{
					Event:  busv1alpha1.PodEvictedEvent,
					Action: busv1alpha1.RestartJobAction,
				},
			},
			Plugins:                 nil,
			RunningEstimate:         nil,
			Queue:                   "default",
			MaxRetry:                0,
			TTLSecondsAfterFinished: nil,
			PriorityClassName:       "",
			MinSuccess:              nil,
		},
		Status: flowv1alpha1.JobTemplateStatus{},
	}
	return jobTemplate
}

func CreateJobFlow(context *TestContext, jobFlow *flowv1alpha1.JobFlow) *flowv1alpha1.JobFlow {
	jobFlowRes, err := CreateJobFlowInner(context, jobFlow)
	Expect(err).NotTo(HaveOccurred(), "failed to create jobFlow %s in namespace %s", jobFlow.Name, jobFlow.Namespace)
	return jobFlowRes
}

func CreateJobFlowInner(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) (*flowv1alpha1.JobFlow, error) {
	jobFlow.Namespace = ctx.Namespace
	err := ctx.JobFlowReconciler.Create(context.TODO(), jobFlow)
	return jobFlow, err
}

func DeleteJobFlow(context *TestContext, jobFlow *flowv1alpha1.JobFlow) *flowv1alpha1.JobFlow {
	jobFlowRes, err := DeleteJobFlowInner(context, jobFlow)
	Expect(err).NotTo(HaveOccurred(), "failed to create jobFlow %s in namespace %s", jobFlow.Name, jobFlow.Namespace)
	return jobFlowRes
}

func DeleteJobFlowInner(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) (*flowv1alpha1.JobFlow, error) {
	jobFlow.Namespace = ctx.Namespace
	err := ctx.JobFlowReconciler.Delete(context.TODO(), jobFlow)
	return jobFlow, err
}

func UpdateJobFlowStatus(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) *flowv1alpha1.JobFlow {
	jobTemplateRes, err := UpdateJobFlowStatusInner(ctx, jobFlow)
	Expect(err).NotTo(HaveOccurred(), "failed to update jobFlow %s in namespace %s", jobFlow.Name, jobFlow.Namespace)
	return jobTemplateRes
}

func UpdateJobFlowStatusInner(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) (*flowv1alpha1.JobFlow, error) {
	jobFlow.Namespace = ctx.Namespace
	err := ctx.JobFlowReconciler.Status().Update(context.TODO(), jobFlow)
	return jobFlow, err
}

func JobFlowExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) wait.ConditionFunc {
	return func() (bool, error) {
		key := types.NamespacedName{
			Namespace: jobFlow.Namespace,
			Name:      jobFlow.Name,
		}
		jobFlowGet := &flowv1alpha1.JobFlow{}
		err := ctx.JobFlowReconciler.Get(context.TODO(), key, jobFlowGet)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

func JobFlowNotExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow) wait.ConditionFunc {
	return func() (bool, error) {
		key := types.NamespacedName{
			Namespace: jobFlow.Namespace,
			Name:      jobFlow.Name,
		}
		jobTemplateGet := &flowv1alpha1.JobTemplate{}
		err := ctx.JobFlowReconciler.Get(context.TODO(), key, jobTemplateGet)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}
}

func GetFlowInstance(jobFlowName string) *flowv1alpha1.JobFlow {
	jobflow := &flowv1alpha1.JobFlow{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: jobFlowName,
		},
		Spec: flowv1alpha1.JobFlowSpec{
			Flows: []flowv1alpha1.Flow{
				{
					Name:      "jobtemplate-a",
					DependsOn: nil,
				},
				{
					Name: "jobtemplate-b",
					DependsOn: &flowv1alpha1.DependsOn{
						Targets: []string{"jobtemplate-a"},
						Probe:   nil,
					},
				},
			},
			JobRetainPolicy: flowv1alpha1.Retain,
		},
		Status: flowv1alpha1.JobFlowStatus{},
	}
	return jobflow
}

func GetFlowInstanceRetainPolicyDelete(jobFlowName string) *flowv1alpha1.JobFlow {
	jobflow := &flowv1alpha1.JobFlow{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: jobFlowName,
		},
		Spec: flowv1alpha1.JobFlowSpec{
			Flows: []flowv1alpha1.Flow{
				{
					Name:      "jobtemplate-a",
					DependsOn: nil,
				},
				{
					Name: "jobtemplate-b",
					DependsOn: &flowv1alpha1.DependsOn{
						Targets: []string{"jobtemplate-a"},
						Probe:   nil,
					},
				},
			},
			JobRetainPolicy: flowv1alpha1.Delete,
		},
		Status: flowv1alpha1.JobFlowStatus{},
	}
	return jobflow
}

func VcJobExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		key := types.NamespacedName{
			Namespace: jobTemplate.Namespace,
			Name:      GetJobName(jobFlow.Name, jobTemplate.Name),
		}
		jobGet := &v1alpha1.Job{}
		err := ctx.JobFlowReconciler.Get(context.TODO(), key, jobGet)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

func VcJobNotExist(ctx *TestContext, jobFlow *flowv1alpha1.JobFlow, jobTemplate *flowv1alpha1.JobTemplate) wait.ConditionFunc {
	return func() (bool, error) {
		key := types.NamespacedName{
			Namespace: jobTemplate.Namespace,
			Name:      GetJobName(jobFlow.Name, jobTemplate.Name),
		}
		jobGet := &v1alpha1.Job{}
		err := ctx.JobFlowReconciler.Get(context.TODO(), key, jobGet)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}
}

func GetJobName(jobFlowName string, jobTemplateName string) string {
	return jobFlowName + "-" + jobTemplateName
}
