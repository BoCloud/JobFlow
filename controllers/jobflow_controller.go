/*


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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"time"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/go-logr/logr"
	batchv1alpha1 "jobflow/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	os_runtime "runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// JobFlowReconciler reconciles a JobFlow object
type JobFlowReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *JobFlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("jobflow", req.NamespacedName)

	// your logic here
	r.Log.Info(fmt.Sprintf("1.%v", req))
	r.Log.Info(fmt.Sprintf("2.%v", os_runtime.NumGoroutine()))

	//根据namespace加载JobFlow
	jobFlow := &batchv1alpha1.JobFlow{}

	err := r.Get(ctx, req.NamespacedName, jobFlow)
	if err != nil {
		//If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			log.Info("No instance found！")
			r.Recorder.Eventf(jobFlow, corev1.EventTypeWarning, "Created", "No jobFlow found！")
			return ctrl.Result{}, nil
		}
		log.Error(err, err.Error())
		r.Recorder.Eventf(jobFlow, corev1.EventTypeWarning, "Created", err.Error())
		return ctrl.Result{}, err
	} else {
		r.Recorder.Eventf(jobFlow, corev1.EventTypeNormal, "Created", "start load JobTemplate!")
	}

	//根据JobFlow的flow依赖加载所有需要的jobTemplate，若无法加载所有需要的jobTemplate则直接返回错误信息
	flowJobTemplateList, err := r.loadJobTemplate(ctx, *jobFlow)
	if err != nil {
		return ctrl.Result{}, err
	}

	//根据依赖顺序下发job。若下发的job没有依赖项，则直接下发。若有依赖则当所有依赖项达到条件后开始下发
	if err = r.deployJob(ctx, flowJobTemplateList); err != nil {
		return ctrl.Result{}, err
	}

	// 声明 finalizer 字段，类型为字符串
	myFinalizerName := "storage.finalizers.tutorial.kubebuilder.io"

	// 通过检查 DeletionTimestamp 字段是否为0 判断资源是否被删除
	if jobFlow.ObjectMeta.DeletionTimestamp.IsZero() {
		// 如果为0 ，则资源未被删除，我们需要检测是否存在 finalizer，如果不存在，则添加，并更新到资源对象中
		if !containsString(jobFlow.ObjectMeta.Finalizers, myFinalizerName) {
			jobFlow.ObjectMeta.Finalizers = append(jobFlow.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), jobFlow); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// 如果不为 0 ，则对象处于删除中
		if containsString(jobFlow.ObjectMeta.Finalizers, myFinalizerName) {
			// 如果存在 finalizer 且与上述声明的 finalizer 匹配，那么执行对应 hook 逻辑
			if err := r.deleteExternalResources(ctx, jobFlow); err != nil {
				// 如果删除失败，则直接返回对应 err，controller 会自动执行重试逻辑
				return ctrl.Result{}, err
			}

			// 如果对应 hook 执行成功，那么清空 finalizers， k8s 删除对应资源
			jobFlow.ObjectMeta.Finalizers = removeString(jobFlow.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), jobFlow); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *JobFlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1alpha1.JobFlow{}).
		Complete(r)
}

//根据JobFlow的flow依赖加载所有需要的jobTemplate，若无法加载所有需要的jobTemplate则直接返回错误信息
func (r *JobFlowReconciler) loadJobTemplate(ctx context.Context, jobFlow batchv1alpha1.JobFlow) (map[string]FlowJobTemplate, error) {
	//加载所有的jobTemplate
	jobTemplateMap := make(map[string]FlowJobTemplate, 0)
	for _, flow := range jobFlow.Spec.Flow {
		namespacedName := types.NamespacedName{
			Namespace: jobFlow.Namespace,
			Name:      flow.Name,
		}
		jobTemplate := &batchv1alpha1.JobTemplate{}
		if err := r.Get(ctx, namespacedName, jobTemplate); err != nil {
			//If no instance is found, it will be returned directly
			if errors.IsNotFound(err) {
				r.Log.Info(fmt.Sprintf("can't found JobTemplate for %v !", flow.Name))
				return nil, errors.NewBadRequest("")
			}
			r.Log.Error(err, err.Error())
			return nil, err
		}
		jobTemplate.ObjectMeta.OwnerReferences = append(jobTemplate.ObjectMeta.OwnerReferences, metav1.OwnerReference{
			APIVersion: jobFlow.APIVersion,
			Kind:       jobFlow.Kind,
			Name:       jobFlow.Name,
		})
		jobTemplate.ObjectMeta.Name = jobFlow.Name + "-" + jobTemplate.ObjectMeta.Name
		jobTemplateMap[flow.Name] = FlowJobTemplate{
			Flow:        flow,
			JobTemplate: jobTemplate,
		}
	}
	return jobTemplateMap, nil
}

const JobFlow = "JobFlow"

type FlowJobTemplate struct {
	Flow        batchv1alpha1.Flow
	JobTemplate *batchv1alpha1.JobTemplate
}

//根据依赖顺序下发job。若下发的job没有依赖项，则直接下发。若有依赖则当所有依赖项达到条件后开始下发
func (r *JobFlowReconciler) deployJob(ctx context.Context, flowJobTemplateMap map[string]FlowJobTemplate) error {
	//部署没有依赖项的job
	for name, flowJobTemplate := range flowJobTemplateMap {
		if len(flowJobTemplate.Flow.DependOn.Target) == 0 {
			if err := r.Create(ctx, flowJobTemplate.JobTemplate); err != nil {
				r.Log.Error(err, "")
				return err
			}
		}
		delete(flowJobTemplateMap, name)
	}
	//部署有依赖项job
	for name, flowJobTemplate := range flowJobTemplateMap {
		flag := true
		for _, targetName := range flowJobTemplate.Flow.DependOn.Target {
			namespacedName := types.NamespacedName{
				Namespace: flowJobTemplate.JobTemplate.Namespace,
				Name:      targetName,
			}
			jobTemplate := &batchv1alpha1.JobTemplate{}
			if err := r.Get(ctx, namespacedName, jobTemplate); err != nil {
				return err
			}
			if jobTemplate.Status.State.Phase != v1alpha1.Completed && jobTemplate.Status.State.Phase != v1alpha1.Completing {
				flag = false
			}
		}
		//依赖项不满足要求则跳过该job
		if !flag {
			continue
		}
		//依赖项满足要求，开始下发该job
		if err := r.Create(ctx, flowJobTemplate.JobTemplate); err != nil {
			return err
		}
		delete(flowJobTemplateMap, name)
	}
	return nil
}

//更新status
func (r *JobFlowReconciler) updateStatus(ctx context.Context, jobFlow batchv1alpha1.JobFlow) error {
	// jobList
	if len(jobFlow.Status.JobList) == 0 {
		jobList := make([]string, 0)
		for _, flow := range jobFlow.Spec.Flow {
			jobList = append(jobList, flow.Name)
		}
		jobFlow.Status.JobList = jobList
	}

	if err := r.Update(ctx, &jobFlow); err != nil {
		return err
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

func (r *JobFlowReconciler) deleteExternalResources(ctx context.Context, jobFlow *batchv1alpha1.JobFlow) error {
	// 删除 guestbook关联的pods
	job := &v1alpha1.Job{}
	job.ObjectMeta.OwnerReferences = append(job.ObjectMeta.OwnerReferences, metav1.OwnerReference{
		APIVersion: jobFlow.APIVersion,
		Kind:       jobFlow.Kind,
		Name:       jobFlow.Name,
	})
	if err := r.DeleteAllOf(ctx, job, client.InNamespace(jobFlow.Namespace)); err != nil {
		r.Log.Error(err, "删除jobFlow关联job失败")
		return err
	}
	return nil
}

//获取所有已经创建的job的信息
func GetAllJob() {

}
