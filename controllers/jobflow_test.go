package controllers

import (
	"reflect"
	"testing"
	"time"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

func TestGetJobNameFunc(t *testing.T) {
	type args struct {
		jobFlowName     string
		jobTemplateName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "GetJobName success case",
			args: args{
				jobFlowName:     "jobFlowA",
				jobTemplateName: "jobTemplateA",
			},
			want: "jobFlowA-jobTemplateA",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJobName(tt.args.jobFlowName, tt.args.jobTemplateName); got != tt.want {
				t.Errorf("getJobName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRunningHistoriesFunc(t *testing.T) {
	type args struct {
		jobStatusList []jobflowv1alpha1.JobStatus
		job           v1alpha1.Job
	}
	startTime := time.Now()
	endTime := startTime.Add(1 * time.Second)
	tests := []struct {
		name string
		args args
		want []jobflowv1alpha1.JobRunningHistory
	}{
		{
			name: "GetRunningHistories success case",
			args: args{
				jobStatusList: []jobflowv1alpha1.JobStatus{
					{
						Name:           "vcJobA",
						State:          v1alpha1.Completed,
						StartTimestamp: v1.Time{Time: startTime},
						EndTimestamp:   v1.Time{Time: endTime},
						RestartCount:   0,
						RunningHistories: []jobflowv1alpha1.JobRunningHistory{
							{
								StartTimestamp: v1.Time{Time: startTime},
								EndTimestamp:   v1.Time{Time: endTime},
								State:          v1alpha1.Completed,
							},
						},
					},
				},
				job: v1alpha1.Job{
					TypeMeta:   v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{Name: "vcJobA"},
					Spec:       v1alpha1.JobSpec{},
					Status: v1alpha1.JobStatus{
						State: v1alpha1.JobState{
							Phase:              v1alpha1.Completed,
							Reason:             "",
							Message:            "",
							LastTransitionTime: v1.Time{},
						},
					},
				},
			},
			want: []jobflowv1alpha1.JobRunningHistory{
				{
					StartTimestamp: v1.Time{Time: startTime},
					EndTimestamp:   v1.Time{Time: endTime},
					State:          v1alpha1.Completed,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRunningHistories(tt.args.jobStatusList, tt.args.job); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRunningHistories() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllJobStatusFunc(t *testing.T) {
	type args struct {
		jobFlow    *jobflowv1alpha1.JobFlow
		allJobList *v1alpha1.JobList
	}
	createJobATime := time.Now()
	jobFlowName := "jobFlowA"
	createJobBTime := createJobATime.Add(time.Second)
	tests := []struct {
		name    string
		args    args
		want    *jobflowv1alpha1.JobFlowStatus
		wantErr bool
	}{
		{
			name: "GetAllJobStatus success case",
			args: args{
				jobFlow: &jobflowv1alpha1.JobFlow{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Name: jobFlowName,
					},
					Spec: jobflowv1alpha1.JobFlowSpec{
						Flows: []jobflowv1alpha1.Flow{
							{
								Name:      "A",
								DependsOn: nil,
							},
							{
								Name: "B",
								DependsOn: &jobflowv1alpha1.DependsOn{
									Targets: []string{"A"},
								},
							},
						},
						JobRetainPolicy: "",
					},
					Status: jobflowv1alpha1.JobFlowStatus{},
				},
				allJobList: &v1alpha1.JobList{
					Items: []v1alpha1.Job{
						{
							TypeMeta: v1.TypeMeta{},
							ObjectMeta: v1.ObjectMeta{
								Name:              "jobFlowA-A",
								CreationTimestamp: v1.Time{Time: createJobATime},
								OwnerReferences: []v1.OwnerReference{{
									APIVersion: "volcano",
									Kind:       JobFlow,
									Name:       jobFlowName,
								}},
							},
							Spec: v1alpha1.JobSpec{},
							Status: v1alpha1.JobStatus{
								State:           v1alpha1.JobState{Phase: v1alpha1.Completed},
								RetryCount:      1,
								RunningDuration: &metav1.Duration{Duration: time.Second},
							},
						},
						{
							TypeMeta: v1.TypeMeta{},
							ObjectMeta: v1.ObjectMeta{
								Name:              "jobFlowA-B",
								CreationTimestamp: v1.Time{Time: createJobBTime},
								OwnerReferences: []v1.OwnerReference{{
									APIVersion: "volcano",
									Kind:       JobFlow,
									Name:       jobFlowName,
								}},
							},
							Spec: v1alpha1.JobSpec{},
							Status: v1alpha1.JobStatus{
								State: v1alpha1.JobState{Phase: v1alpha1.Running},
							},
						},
					},
				},
			},
			want: &jobflowv1alpha1.JobFlowStatus{
				PendingJobs:    []string{},
				RunningJobs:    []string{"jobFlowA-B"},
				FailedJobs:     []string{},
				CompletedJobs:  []string{"jobFlowA-A"},
				TerminatedJobs: []string{},
				UnKnowJobs:     []string{},
				JobStatusList: []jobflowv1alpha1.JobStatus{
					{
						Name:           "jobFlowA-A",
						State:          v1alpha1.Completed,
						StartTimestamp: metav1.Time{Time: createJobATime},
						EndTimestamp:   metav1.Time{Time: createJobATime.Add(time.Second)},
						RestartCount:   1,
						RunningHistories: []jobflowv1alpha1.JobRunningHistory{
							{
								StartTimestamp: metav1.Time{},
								EndTimestamp:   metav1.Time{},
								State:          v1alpha1.Completed,
							},
						},
					},
					{
						Name:           "jobFlowA-B",
						State:          v1alpha1.Running,
						StartTimestamp: metav1.Time{Time: createJobBTime},
						EndTimestamp:   metav1.Time{},
						RestartCount:   0,
						RunningHistories: []jobflowv1alpha1.JobRunningHistory{
							{
								StartTimestamp: metav1.Time{},
								EndTimestamp:   metav1.Time{},
								State:          v1alpha1.Running,
							},
						},
					},
				},
				Conditions: map[string]jobflowv1alpha1.Condition{
					"jobFlowA-A": {
						Phase:           v1alpha1.Completed,
						CreateTimestamp: metav1.Time{Time: createJobATime},
						RunningDuration: &v1.Duration{Duration: time.Second},
					},
					"jobFlowA-B": {
						Phase:           v1alpha1.Running,
						CreateTimestamp: metav1.Time{Time: createJobBTime},
					},
				},
				State: jobflowv1alpha1.State{Phase: jobflowv1alpha1.Running},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAllJobStatus(tt.args.jobFlow, tt.args.allJobList)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAllJobStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got.JobStatusList[0].RunningHistories[0].StartTimestamp = metav1.Time{}
			got.JobStatusList[1].RunningHistories[0].StartTimestamp = metav1.Time{}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllJobStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}
