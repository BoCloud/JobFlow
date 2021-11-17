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

	batchv1alpha1 "jobflow/api/v1alpha1"
	jobflowv1alpha1 "jobflow/api/v1alpha1"
	"jobflow/utils"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

// JobTemplateReconciler reconciles a Job object
type JobTemplateReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=batch.volcano.sh,resources=jobtemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.volcano.sh,resources=jobtemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.volcano.sh,resources=jobtemplates/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.volcano.sh,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Job object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *JobTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.Info("start jobTemplate Reconcile..........")
	klog.Info(fmt.Sprintf("event for jobTemplate: %v", req.Name))
	// your logic here
	scheduledResult := ctrl.Result{}

	//根据namespace加载JobTemplate
	jobTemplate := &jobflowv1alpha1.JobTemplate{}
	time.Sleep(time.Second)
	err := r.Get(ctx, req.NamespacedName, jobTemplate)
	if err != nil {
		//If no instance is found, it will be returned directly
		if errors.IsNotFound(err) {
			klog.Info(fmt.Sprintf("not fount jobTemplate : %v", req.Name))
			return scheduledResult, nil
		}
		klog.Error(err, err.Error())
		r.Recorder.Eventf(jobTemplate, corev1.EventTypeWarning, "Created", err.Error())
		return scheduledResult, err
	}
	//查询根据JobTemplate创建的job
	jobList := &v1alpha1.JobList{}
	err = r.List(ctx, jobList)
	if err != nil {
		klog.Error(err, "")
		return scheduledResult, err
	}
	filterJobList := make([]v1alpha1.Job, 0)
	for _, item := range jobList.Items {
		if item.Annotations[utils.CreateByJobTemplate] == utils.GetConnectionOfJobAndJobTemplate(req.Namespace, req.Name) {
			filterJobList = append(filterJobList, item)
		}
	}
	if len(filterJobList) == 0 {
		return scheduledResult, err
	}
	jobListName := make([]string, 0)
	for _, job := range filterJobList {
		jobListName = append(jobListName, job.Name)
	}
	jobTemplate.Status.JobDependsOnList = jobListName
	//更新
	if err := r.Status().Update(ctx, jobTemplate); err != nil {
		klog.Error(err, "update error!")
		return scheduledResult, err
	}
	klog.Info("end jobTemplate Reconcile..........")
	return scheduledResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1alpha1.JobTemplate{}).
		Watches(&source.Kind{Type: &v1alpha1.Job{}}, handler.Funcs{CreateFunc: jobCreateHandler}).
		Complete(r)
}

func jobCreateHandler(e event.CreateEvent, w workqueue.RateLimitingInterface) {
	if e.Object.GetAnnotations()[utils.CreateByJobTemplate] != "" {
		nameNamespace := strings.Split(e.Object.GetAnnotations()[utils.CreateByJobTemplate], ".")
		namespace, name := nameNamespace[0], nameNamespace[1]
		w.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
	}
}
