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
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sort"
	"strings"
	"time"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
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
	klog.Info("start jobFlow Reconcile..........")
	klog.Info(fmt.Sprintf("req.%v", req))

	scheduledResult := ctrl.Result{}
	//根据namespace加载JobFlow
	jobFlow := &jobflowv1alpha1.JobFlow{}
	time.Sleep(time.Second)
	err := r.Get(ctx, req.NamespacedName, jobFlow)
	if err != nil {
		//If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			klog.Info(fmt.Sprintf("not fount jobFlow : %v", req.Name))
			return scheduledResult, nil
		}
		klog.Error(err, err.Error())
		r.Recorder.Eventf(jobFlow, corev1.EventTypeWarning, "Created", err.Error())
		return scheduledResult, err
	}

	//根据依赖顺序下发job。若下发的job没有依赖项，则直接下发。若有依赖则当所有依赖项达到条件后开始下发
	if err = r.deployJob(ctx, *jobFlow); err != nil {
		klog.Error(err, "")
		return scheduledResult, err
	}

	//更新status
	if err = r.updateStatus(ctx, jobFlow); err != nil {
		klog.Error(err, "更新jobFlow status错误")
		return scheduledResult, err
	}
	klog.Info("end  jobFlow   Reconcile........")
	return scheduledResult, nil
}

func getJobName(jobFlowName string, jobTemplateName string) string {
	return jobFlowName + "-" + jobTemplateName
}

const (
	JobFlow = "JobFlow"
)

//根据依赖顺序下发job。若下发的job没有依赖项，则直接下发。若有依赖则当所有依赖项达到条件后开始下发
func (r *JobFlowReconciler) deployJob(ctx context.Context, jobFlow jobflowv1alpha1.JobFlow) error {
	//根据flow加载对应的jobTemplate并下发
	for _, flow := range jobFlow.Spec.Flows {
		//查询该job是否已经下发
		job := &v1alpha1.Job{}
		jobName := getJobName(jobFlow.Name, flow.Name)
		namespacedNameJob := types.NamespacedName{
			Namespace: jobFlow.Namespace,
			Name:      jobName,
		}
		if err := r.Get(ctx, namespacedNameJob, job); err != nil {
			if errors.IsNotFound(err) {
				//没有下发则判断该vcjob的依赖项是否符合要求
				if len(flow.DependsOn.Target) == 0 {
					//加载对应的jobTemplate
					jobTemplate := &jobflowv1alpha1.JobTemplate{}
					namespacedNameTemplate := types.NamespacedName{
						Namespace: jobFlow.Namespace,
						Name:      flow.Name,
					}
					if err = r.Get(ctx, namespacedNameTemplate, jobTemplate); err != nil {
						klog.Error(err, "未查询到该jobTemplate！")
						return err
					}
					job = &v1alpha1.Job{
						ObjectMeta: metav1.ObjectMeta{
							Name:        jobName,
							Namespace:   jobFlow.Namespace,
							Annotations: map[string]string{utils.CreateByJobTemplate: utils.GetCreateByJobTemplateValue(jobFlow.Namespace, flow.Name)},
						},
						Spec:   jobTemplate.Spec,
						Status: v1alpha1.JobStatus{},
					}
					if err := controllerutil.SetControllerReference(&jobFlow, job, r.Scheme); err != nil {
						return err
					}
					if err = r.Create(ctx, job); err != nil {
						if errors.IsAlreadyExists(err) {
							continue
						}
						return err
					}
					r.Recorder.Eventf(&jobFlow, corev1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v!", job.Name))
				} else {
					//查询依赖项时候符合要求
					flag := true
					for _, targetName := range flow.DependsOn.Target {
						job = &v1alpha1.Job{}
						targetJobName := getJobName(jobFlow.Name, targetName)
						namespacedName := types.NamespacedName{
							Namespace: jobFlow.Namespace,
							Name:      targetJobName,
						}
						if err = r.Get(ctx, namespacedName, job); err != nil {
							if err != nil {
								if errors.IsNotFound(err) {
									klog.Info(fmt.Sprintf("No %v Job found！", namespacedName.Name))
									flag = false
									break
								}
								return err
							}
						}
						if job.Status.State.Phase != v1alpha1.Completed {
							flag = false
						}
					}
					//依赖条件满足时
					if flag {
						//加载对应的jobTemplate
						jobTemplate := &jobflowv1alpha1.JobTemplate{}
						namespacedNameTemplate := types.NamespacedName{
							Namespace: jobFlow.Namespace,
							Name:      flow.Name,
						}
						if err = r.Get(ctx, namespacedNameTemplate, jobTemplate); err != nil {
							klog.Error(err, "未查询到该jobTemplate！")
							return err
						}
						job = &v1alpha1.Job{
							ObjectMeta: metav1.ObjectMeta{
								Name:        jobName,
								Namespace:   jobFlow.Namespace,
								Annotations: map[string]string{utils.CreateByJobTemplate: utils.GetCreateByJobTemplateValue(jobFlow.Namespace, flow.Name)},
							},
							Spec:   jobTemplate.Spec,
							Status: v1alpha1.JobStatus{},
						}
						if err = controllerutil.SetControllerReference(&jobFlow, job, r.Scheme); err != nil {
							return err
						}
						if err = r.Create(ctx, job); err != nil {
							if errors.IsAlreadyExists(err) {
								break
							}
							return err
						}
						r.Recorder.Eventf(&jobFlow, corev1.EventTypeNormal, "Created", fmt.Sprintf("create a job named %v!", job.Name))
					}
				}
				continue
			}
			return err
		}
	}
	return nil
}

//更新status
func (r *JobFlowReconciler) updateStatus(ctx context.Context, jobFlow *jobflowv1alpha1.JobFlow) error {
	klog.Info(fmt.Sprintf("开始更新jobFlow status! jobFlowName: %v, jobFlowNamespace: %v ", jobFlow.Name, jobFlow.Namespace))
	jobFlowStatus, err := r.getAllJobStatus(ctx, jobFlow)
	if err != nil {
		return err
	}
	jobFlow.Status = *jobFlowStatus
	jobFlow.CreationTimestamp = metav1.Time{}
	jobFlow.UID = ""
	if err = r.Status().Update(ctx, jobFlow); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// getAllJobStatus 获取所有已经创建的job的信息
func (r *JobFlowReconciler) getAllJobStatus(ctx context.Context, jobFlow *jobflowv1alpha1.JobFlow) (*jobflowv1alpha1.JobFlowStatus, error) {
	allJobList := new(v1alpha1.JobList)
	err := r.List(ctx, allJobList)
	if err != nil {
		klog.Error(err, "")
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
	jobList := make([]string, 0)
	state := new(jobflowv1alpha1.State)
	for _, flow := range jobFlow.Spec.Flows {
		jobList = append(jobList, getJobName(jobFlow.Name, flow.Name))
	}
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
			Phase:           job.Status.State.Phase,
			CreateTime:      job.CreationTimestamp,
			RunningDuration: job.Status.RunningDuration,
			TaskStatusCount: job.Status.TaskStatusCount,
		}
	}
	if jobFlow.DeletionTimestamp != nil {
		state.Phase = jobflowv1alpha1.Terminating
	} else {
		if len(jobList) != len(CompletedJobs) {
			if len(FailedJobs) > 0 {
				state.Phase = jobflowv1alpha1.Failed
			} else if len(runningJobs) > 0 || len(CompletedJobs) > 0 {
				state.Phase = jobflowv1alpha1.Running
			} else {
				state.Phase = jobflowv1alpha1.Pending
			}
		} else {
			state.Phase = jobflowv1alpha1.Succeed
		}
	}

	sort.Slice(jobList, func(i, j int) bool {
		return jobList[i] < jobList[j]
	})
	jobFlowStatus := jobflowv1alpha1.JobFlowStatus{
		PendingJobs:    pendingJobs,
		RunningJobs:    runningJobs,
		FailedJobs:     FailedJobs,
		CompletedJobs:  CompletedJobs,
		TerminatedJobs: TerminatedJobs,
		UnKnowJobs:     UnKnowJobs,
		JobList:        jobList,
		Conditions:     conditions,
		State:          *state,
	}
	return &jobFlowStatus, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobFlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jobflowv1alpha1.JobFlow{}).
		Watches(&source.Kind{Type: &v1alpha1.Job{}}, handler.Funcs{UpdateFunc: r.jobUpdateHandler}).
		Complete(r)
}

func (r *JobFlowReconciler) jobUpdateHandler(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	references := e.ObjectOld.GetOwnerReferences()
	for _, owner := range references {
		if owner.Kind == "JobFlow" && strings.Contains(owner.APIVersion, "volcano") {
			klog.Info(fmt.Sprintf("监听到job的更新事件！jobName: %v, jobFlowName: %v, nameSpace: %v", e.ObjectOld.GetName(), owner.Name, e.ObjectOld.GetNamespace()))
			q.AddRateLimited(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: owner.Name, Namespace: e.ObjectOld.GetNamespace()},
			})
		}
	}
}
