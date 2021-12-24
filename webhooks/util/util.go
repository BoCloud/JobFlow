package util

import (
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// ToAdmissionResponse updates the admission response with the input error.
func ToAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	if err != nil {
		klog.Error(err)
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
}
