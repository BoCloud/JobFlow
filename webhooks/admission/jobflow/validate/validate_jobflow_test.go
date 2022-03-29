package validate

import (
	"jobflow/webhooks/util"
	"strings"
	"testing"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
)

func TestValidateJobCreate(t *testing.T) {
	namespace := "test"

	testCases := []struct {
		Name           string
		JobFlow        jobflowv1alpha1.JobFlow
		ExpectErr      bool
		reviewResponse v1beta1.AdmissionResponse
		ret            string
	}{
		{
			Name: "validate valid-jobFlow",
			JobFlow: jobflowv1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-jobFlow",
					Namespace: namespace,
				},
				Spec: jobflowv1alpha1.JobFlowSpec{
					Flows: []jobflowv1alpha1.Flow{
						{
							Name: "job-a",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{},
							},
						},
						{
							Name: "job-b",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{"job-a"},
							},
						},
						{
							Name: "job-c",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{"job-b"},
							},
						},
					},
					JobRetainPolicy: "succeed",
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "",
			ExpectErr:      false,
		},
		// duplicate jobTemplate name
		{
			Name: "duplicate-job-jobFlow",
			JobFlow: jobflowv1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-job-jobFlow",
					Namespace: namespace,
				},
				Spec: jobflowv1alpha1.JobFlowSpec{
					Flows: []jobflowv1alpha1.Flow{
						{
							Name: "job-a",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{},
							},
						},
						{
							Name: "job-b",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{},
							},
						},
						{
							Name: "job-b",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{},
							},
						},
					},
					JobRetainPolicy: "succeed",
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicated template name job-b",
			ExpectErr:      true,
		},
		// dependsOn illegal
		{
			Name: "dependsOn-illegal-jobFlow",
			JobFlow: jobflowv1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dependsOn-illegal-jobFlow",
					Namespace: namespace,
				},
				Spec: jobflowv1alpha1.JobFlowSpec{
					Flows: []jobflowv1alpha1.Flow{
						{
							Name: "job-a",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{},
							},
						},
						{
							Name: "job-b",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{"job-m"},
							},
						},
					},
					JobRetainPolicy: "succeed",
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "cannot find the template: job-m",
			ExpectErr:      true,
		},
		// bad-dag-jobFlow
		{
			Name: "bad-dag-jobFlow",
			JobFlow: jobflowv1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bad-dag-jobFlow",
					Namespace: namespace,
				},
				Spec: jobflowv1alpha1.JobFlowSpec{
					Flows: []jobflowv1alpha1.Flow{
						{
							Name: "job-a",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{"job-c", "job-b"},
							},
						},
						{
							Name: "job-b",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{"job-a", "job-c"},
							},
						},
						{
							Name: "job-c",
							DependsOn: &jobflowv1alpha1.DependsOn{
								Targets: []string{"job-b", "job-a"},
							},
						},
					},
					JobRetainPolicy: "succeed",
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "find bad dependency, please check the dependencies of your templates",
			ExpectErr:      true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			ret := validateJobFlowCreate(&testCase.JobFlow)
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
