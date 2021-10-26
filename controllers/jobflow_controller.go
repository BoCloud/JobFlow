/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"jobflow/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
	batchv1alpha1 "volcano.sh/apis/pkg/client/clientset/versioned/typed/batch/v1alpha1"
)

// JobFlowReconciler reconciles a JobFlow object
type JobFlowReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=batch.volcano.sh,resources=jobflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.volcano.sh,resources=jobflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.volcano.sh,resources=jobflows/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the JobFlow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *JobFlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	//log.Log.Info(fmt.Sprintf("req.%v", req))
	//log.Log.Info(fmt.Sprintf("goruntime.NumGoroutine():%v", goruntime.NumGoroutine()))

	scheduledResult := ctrl.Result{}
	//初始化vcclient
	vcclient := utils.VcClient
	//根据namespace加载JobFlow
	jobFlow := &jobflowv1alpha1.JobFlow{}
	err := r.Get(ctx, req.NamespacedName, jobFlow)
	if err != nil {
		//If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			return scheduledResult, nil
		}
		log.Log.Error(err, err.Error())
		r.Recorder.Eventf(jobFlow, corev1.EventTypeWarning, "Created", err.Error())
		return scheduledResult, err
	}

	//根据JobFlow的flow依赖加载所有需要的jobTemplate，若无法加载所有需要的jobTemplate则直接返回错误信息
	flowJobList, jobList, err := r.loadJobTemplate(ctx, *jobFlow)
	if err != nil {
		return scheduledResult, err
	}
	//根据依赖顺序下发job。若下发的job没有依赖项，则直接下发。若有依赖则当所有依赖项达到条件后开始下发
	if err = r.deployJob(ctx, flowJobList, *jobFlow, vcclient); err != nil {
		log.Log.Error(err, "")
		return scheduledResult, err
	}
	//更新status
	fmt.Println("开始更新status")
	fmt.Println(time.Now())
	if err = r.updateStatus(ctx, jobFlow, jobList, vcclient); err != nil {
		log.Log.Error(err, "更新status错误")
		return scheduledResult, err
	}
	// 声明 finalizer 字段，类型为字符串
	myFinalizerName := "storage.finalizers.tutorial.kubebuilder.io"

	// 通过检查 DeletionTimestamp 字段是否为0 判断资源是否被删除
	if jobFlow.ObjectMeta.DeletionTimestamp.IsZero() {
		// 如果为0 ，则资源未被删除，我们需要检测是否存在 finalizer，如果不存在，则添加，并更新到资源对象中
		if !containsString(jobFlow.ObjectMeta.Finalizers, myFinalizerName) {
			jobFlow.ObjectMeta.Finalizers = append(jobFlow.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), jobFlow); err != nil {
				return scheduledResult, err
			}
		}
	} else {
		// 如果不为 0 ，则对象处于删除中
		if containsString(jobFlow.ObjectMeta.Finalizers, myFinalizerName) {
			// 如果存在 finalizer 且与上述声明的 finalizer 匹配，那么执行对应 hook 逻辑
			if err := r.deleteExternalResources(ctx, jobFlow, vcclient); err != nil {
				// 如果删除失败，则直接返回对应 err，controller 会自动执行重试逻辑
				return scheduledResult, err
			}

			// 如果对应 hook 执行成功，那么清空 finalizers， k8s 删除对应资源
			jobFlow.ObjectMeta.Finalizers = removeString(jobFlow.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), jobFlow); err != nil {
				return scheduledResult, err
			}
		}
		return scheduledResult, nil
	}
	return scheduledResult, nil
}

//根据JobFlow的flow依赖加载所有需要的jobTemplate，若无法加载所有需要的jobTemplate则直接返回错误信息
func (r *JobFlowReconciler) loadJobTemplate(ctx context.Context, jobFlow jobflowv1alpha1.JobFlow) (map[string]FlowJobTemplate, []string, error) {
	//加载所有的jobTemplate
	flowJobList := make(map[string]FlowJobTemplate, 0)
	for _, flow := range jobFlow.Spec.Flows {
		namespacedName := types.NamespacedName{
			Namespace: jobFlow.Namespace,
			Name:      flow.Name,
		}
		jobTemplate := &jobflowv1alpha1.JobTemplate{}
		if err := r.Get(ctx, namespacedName, jobTemplate); err != nil {
			//If no instance is found, it will be returned directly
			if errors.IsNotFound(err) {
				log.Log.Error(err, fmt.Sprintf("can't found Job for %v !", flow.Name))
				return nil, nil, err
			}
			log.Log.Error(err, err.Error())
			return nil, nil, err
		}
		jobName := getJobName(jobFlow.Name, jobTemplate.ObjectMeta.Name)
		targets := make([]string, 0)
		for i := range flow.DependsOn.Target {
			targets = append(targets, getJobName(jobFlow.Name, flow.DependsOn.Target[i]))
		}
		job := &v1alpha1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: jobFlow.Namespace,
			},
			Spec:   jobTemplate.Spec,
			Status: v1alpha1.JobStatus{},
		}
		newFlow := &jobflowv1alpha1.Flow{
			Name: jobName,
			DependsOn: &jobflowv1alpha1.DependsOn{
				Target: targets,
			},
		}
		flowJobList[jobName] = FlowJobTemplate{
			Flow: *newFlow,
			Job:  job,
		}
	}
	//增加jobFlow.status.jobList
	jobList := make([]string, 0)
	for jobName := range flowJobList {
		jobList = append(jobList, jobName)
	}
	return flowJobList, jobList, nil
}

func getJobName(jobFlowName string, jobTemplateName string) string {
	return jobFlowName + "-" + jobTemplateName
}

const (
	JobFlow = "JobFlow"
	Job     = "Job"
)

type FlowJobTemplate struct {
	Flow jobflowv1alpha1.Flow
	Job  *v1alpha1.Job
}

//根据依赖顺序下发job。若下发的job没有依赖项，则直接下发。若有依赖则当所有依赖项达到条件后开始下发
func (r *JobFlowReconciler) deployJob(ctx context.Context, flowJobMap map[string]FlowJobTemplate, jobFlow jobflowv1alpha1.JobFlow, vcclient *batchv1alpha1.BatchV1alpha1Client) error {
	//部署没有依赖项的job
	for name, flowJob := range flowJobMap {
		if len(flowJob.Flow.DependsOn.Target) == 0 {
			//add ownerReferences
			if err := controllerutil.SetControllerReference(&jobFlow, flowJob.Job, r.Scheme); err != nil {
				return err
			}
			//create job
			if err := r.Create(ctx, flowJob.Job); err != nil {
				if errors.IsAlreadyExists(err) {
					continue
				}
				return err
			}
			r.Recorder.Eventf(&jobFlow, corev1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v!", flowJob.Job.Name))
			delete(flowJobMap, name)
			continue
		} else {
			//部署有依赖项job
			flag := true
			for _, targetName := range flowJob.Flow.DependsOn.Target {
				job := &v1alpha1.Job{}
				namespacedName := types.NamespacedName{
					Namespace: flowJob.Job.Namespace,
					Name:      targetName,
				}
				if err := r.Get(ctx, namespacedName, job); err != nil {
					if err != nil {
						if errors.IsNotFound(err) {
							log.Log.Info(fmt.Sprintf("No %v Job found！", namespacedName.Name))
							flag = false
							continue
						} else {
							return err
						}
					}
				}
				//job, err := vcclient.Jobs(flowJob.Job.Namespace).Get(ctx, getJobName(jobFlow.Name, targetName), metav1.GetOptions{})
				//if err != nil {
				//	if errors.IsNotFound(err) {
				//		log.Log.Info("No Job found！")
				//		flag = false
				//		continue
				//	} else {
				//		return err
				//	}
				//}
				if job.Status.State.Phase != v1alpha1.Completed && job.Status.State.Phase != v1alpha1.Completing {
					flag = false
				}
			}
			//依赖项不满足要求则跳过该job
			if !flag {
				continue
			}
			//依赖项满足要求，开始下发该job
			if err := controllerutil.SetControllerReference(&jobFlow, flowJob.Job, r.Scheme); err != nil {
				return err
			}
			if err := r.Create(ctx, flowJob.Job); err != nil {
				if errors.IsAlreadyExists(err) {
					continue
				}
				return err
			}
			//if _, err := vcclient.Jobs(flowJob.Job.Namespace).Create(ctx, flowJob.Job, metav1.CreateOptions{}); err != nil {
			//	if errors.IsAlreadyExists(err) {
			//		continue
			//	}
			//	log.Log.Error(err, err.Error())
			//	return err
			//}
			r.Recorder.Eventf(&jobFlow, corev1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v!", flowJob.Job.Name))
			delete(flowJobMap, name)
		}
	}
	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func (r *JobFlowReconciler) deleteExternalResources(ctx context.Context, jobFlow *jobflowv1alpha1.JobFlow, vcclient *batchv1alpha1.BatchV1alpha1Client) error {
	// 删除 jobFlow关联的所有job
	for _, flow := range jobFlow.Spec.Flows {
		if err := vcclient.Jobs(jobFlow.Namespace).Delete(ctx, getJobName(jobFlow.Name, flow.Name), metav1.DeleteOptions{}); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.Log.Error(err, "")
			return err
		}

	}
	return nil
}

//更新status
func (r *JobFlowReconciler) updateStatus(ctx context.Context, jobFlow *jobflowv1alpha1.JobFlow, jobList []string, vcclient *batchv1alpha1.BatchV1alpha1Client) error {
	jobFlowStatus, err := r.getAllJobStatus(ctx, jobFlow, jobList, vcclient)
	if err != nil {
		return err
	}
	jobFlow.Status = *jobFlowStatus
	jobFlow.CreationTimestamp = metav1.Time{}
	jobFlow.UID = ""
	if err = r.Status().Update(ctx, jobFlow); err != nil {
		return err
	}
	return nil
}

// getAllJobStatus 获取所有已经创建的job的信息
func (r *JobFlowReconciler) getAllJobStatus(ctx context.Context, jobFlow *jobflowv1alpha1.JobFlow, jobList []string, vcclient *batchv1alpha1.BatchV1alpha1Client) (*jobflowv1alpha1.JobFlowStatus, error) {
	allJobList, err := vcclient.Jobs(jobFlow.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Log.Error(err, "")
		return nil, err
	}
	jobListRes := make([]v1alpha1.Job, 0)
	for _, job := range allJobList.Items {
		for _, reference := range job.OwnerReferences {
			if reference.Kind == JobFlow && strings.Contains(reference.APIVersion, "volcano") && reference.Name == jobFlow.Name {
				jobListRes = append(jobListRes, job)
			}
		}
	}
	conditions := make(map[string]jobflowv1alpha1.Condition)
	pendingJobs := make([]string, 0)
	runningJobs := make([]string, 0)
	FailedJobs := make([]string, 0)
	CompletedJobs := make([]string, 0)
	TerminatedJobs := make([]string, 0)
	UnKnowJobs := make([]string, 0)
	for _, job := range jobListRes {
		switch job.Status.State.Phase {
		case v1alpha1.Pending:
			pendingJobs = append(pendingJobs, job.Name)
		case v1alpha1.Running:
			runningJobs = append(runningJobs, job.Name)
		case v1alpha1.Completing:
			CompletedJobs = append(CompletedJobs, job.Name)
		case v1alpha1.Completed:
			CompletedJobs = append(CompletedJobs, job.Name)
		case v1alpha1.Terminating:
			TerminatedJobs = append(TerminatedJobs, job.Name)
		case v1alpha1.Terminated:
			TerminatedJobs = append(TerminatedJobs, job.Name)
		case v1alpha1.Failed:
			FailedJobs = append(FailedJobs, job.Name)
		default:
			UnKnowJobs = append(UnKnowJobs, job.Name)
		}

		conditions[job.Name] = jobflowv1alpha1.Condition{
			Phase:           &job.Status.State.Phase,
			CreateTime:      &job.CreationTimestamp,
			RunningDuration: job.Status.RunningDuration,
			TaskStatusCount: job.Status.TaskStatusCount,
		}
	}

	jobFlowStatus := jobflowv1alpha1.JobFlowStatus{
		PendingJobs:    pendingJobs,
		RunningJobs:    runningJobs,
		FailedJobs:     FailedJobs,
		CompletedJobs:  CompletedJobs,
		TerminatedJobs: TerminatedJobs,
		UnKnowJobs:     UnKnowJobs,
		JobList:        jobList,
		Conditions:     conditions,
	}
	return &jobFlowStatus, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobFlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jobflowv1alpha1.JobFlow{}).
		Owns(&v1alpha1.Job{}).
		Watches(&source.Kind{Type: &v1alpha1.Job{}}, handler.Funcs{UpdateFunc: r.jobUpdateHandler}).
		Complete(r)
}

func (r *JobFlowReconciler) jobUpdateHandler(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	references := e.ObjectOld.GetOwnerReferences()
	for _, owner := range references {
		if owner.Kind == "JobFlow" && strings.Contains(owner.APIVersion, "volcano") {
			q.AddRateLimited(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: owner.Name, Namespace: e.ObjectOld.GetNamespace()},
			})
		}
	}
}
