package validate

import (
	"strings"
	"testing"

	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	busv1alpha1 "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
	"jobflow/webhooks/util"
)

func TestValidateJobCreate(t *testing.T) {
	namespace := "test"
	var invTTL int32 = -1
	var policyExitCode int32 = -1

	testCases := []struct {
		Name           string
		JobTemplate    jobflowv1alpha1.JobTemplate
		ExpectErr      bool
		reviewResponse v1beta1.AdmissionResponse
		ret            string
	}{
		{
			Name: "validate valid-jobTemplate",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-jobTemplate",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.PodEvictedEvent,
							Action: busv1alpha1.RestartTaskAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "",
			ExpectErr:      false,
		},
		// duplicate task name
		{
			Name: "duplicate-task-jobTemplate",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-task-jobTemplate",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "duplicated-task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
						{
							Name:     "duplicated-task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicated task name duplicated-task-1",
			ExpectErr:      true,
		},
		// Duplicated Policy Event
		{
			Name: "jobTemplate-policy-duplicated",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobTemplate-policy-duplicated",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.PodFailedEvent,
							Action: busv1alpha1.AbortJobAction,
						},
						{
							Event:  busv1alpha1.PodFailedEvent,
							Action: busv1alpha1.RestartJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicate",
			ExpectErr:      true,
		},
		// Min Available illegal
		{
			Name: "Min Available illegal",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobTemplate-min-illegal",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 2,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "job 'minAvailable' should not be greater than total replicas in tasks",
			ExpectErr:      true,
		},
		// ttl-illegal
		{
			Name: "jobTemplate-ttl-illegal",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobTemplate-ttl-illegal",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					TTLSecondsAfterFinished: &invTTL,
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "'ttlSecondsAfterFinished' cannot be less than zero",
			ExpectErr:      true,
		},
		// min-MinAvailable less than zero
		{
			Name: "minAvailable-lessThanZero",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "minAvailable-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: -1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "job 'minAvailable' must be >= 0",
			ExpectErr:      true,
		},
		// maxretry less than zero
		{
			Name: "maxretry-lessThanZero",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "maxretry-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					MaxRetry:     -1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "'maxRetry' cannot be less than zero",
			ExpectErr:      true,
		},
		// no task specified in the job
		{
			Name: "no-task",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-task",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks:        []v1alpha1.TaskSpec{},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "no task specified in job spec",
			ExpectErr:      true,
		},
		// replica set less than zero
		{
			Name: "replica-lessThanZero",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: -1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "'replicas' < 0 in task: task-1;",
			ExpectErr:      true,
		},
		// task name error
		{
			Name: "nonDNS-task",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "Task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret: "[a DNS-1123 label must consist of lower case alphanumeric characters or '-', and " +
				"must start and end with an alphanumeric character (e.g. 'my-name',  " +
				"or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')];",
			ExpectErr: true,
		},
		// Policy Event with exit code
		{
			Name: "jobTemplate-policy-withExitCode",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobTemplate-policy-withExitCode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:    busv1alpha1.PodFailedEvent,
							Action:   busv1alpha1.AbortJobAction,
							ExitCode: &policyExitCode,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "must not specify event and exitCode simultaneously",
			ExpectErr:      true,
		},
		// Both policy event and exit code are nil
		{
			Name: "policy-noEvent-noExCode",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "policy-noEvent-noExCode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Action: busv1alpha1.AbortJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "either event and exitCode should be specified",
			ExpectErr:      true,
		},
		// invalid policy event
		{
			Name: "invalid-policy-event",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-policy-event",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.Event("someFakeEvent"),
							Action: busv1alpha1.AbortJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "invalid policy event",
			ExpectErr:      true,
		},
		// invalid policy action
		{
			Name: "invalid-policy-action",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-policy-action",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.PodEvictedEvent,
							Action: busv1alpha1.Action("someFakeAction"),
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "invalid policy action",
			ExpectErr:      true,
		},
		// policy exit-code zero
		{
			Name: "policy-extcode-zero",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "policy-extcode-zero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Action: busv1alpha1.AbortJobAction,
							ExitCode: func(i int32) *int32 {
								return &i
							}(int32(0)),
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "0 is not a valid error code",
			ExpectErr:      true,
		},
		// duplicate policy exit-code
		{
			Name: "duplicate-exitcode",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-exitcode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							ExitCode: func(i int32) *int32 {
								return &i
							}(int32(1)),
						},
						{
							ExitCode: func(i int32) *int32 {
								return &i
							}(int32(1)),
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicate exitCode 1",
			ExpectErr:      true,
		},
		// Policy with any event and other events
		{
			Name: "jobTemplate-policy-withExitCode",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobTemplate-policy-withExitCode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.AnyEvent,
							Action: busv1alpha1.AbortJobAction,
						},
						{
							Event:  busv1alpha1.PodFailedEvent,
							Action: busv1alpha1.RestartJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "if there's * here, no other policy should be here",
			ExpectErr:      true,
		},
		// invalid mount volume
		{
			Name: "invalid-mount-volume",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-mount-volume",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.AnyEvent,
							Action: busv1alpha1.AbortJobAction,
						},
					},
					Volumes: []v1alpha1.VolumeSpec{
						{
							MountPath: "",
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "mountPath is required;",
			ExpectErr:      true,
		},
		// duplicate mount volume
		{
			Name: "duplicate-mount-volume",
			JobTemplate: jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-mount-volume",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  busv1alpha1.AnyEvent,
							Action: busv1alpha1.AbortJobAction,
						},
					},
					Volumes: []v1alpha1.VolumeSpec{
						{
							MountPath:       "/var",
							VolumeClaimName: "pvc1",
						},
						{
							MountPath:       "/var",
							VolumeClaimName: "pvc2",
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicated mountPath: /var;",
			ExpectErr:      true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			ret := validateJobTemplateCreate(&testCase.JobTemplate)
			testCase.reviewResponse = *util.ToAdmissionResponse(ret)
			//fmt.Printf("test-case name:%s, ret:%v  testCase.reviewResponse:%v \n", testCase.Name, ret,testCase.reviewResponse)
			if testCase.ExpectErr == true && ret == nil {
				t.Errorf("Expect error msg :%s, but got nil.", testCase.ret)
			}
			if testCase.ExpectErr == true && testCase.reviewResponse.Allowed != false {
				t.Errorf("Expect Allowed as false but got true.")
			}
			if testCase.ExpectErr == true && ret != nil && !strings.Contains(ret.Error(), testCase.ret) {
				t.Errorf("Expect error msg :%s, but got diff error %v", testCase.ret, ret)
			}

			if testCase.ExpectErr == false && ret != nil {
				t.Errorf("Expect no error, but got error %v", ret)
			}
			if testCase.ExpectErr == false && testCase.reviewResponse.Allowed != true {
				t.Errorf("Expect Allowed as true but got false. %v", testCase.reviewResponse)
			}
		})
	}
}
