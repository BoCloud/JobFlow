package router

import (
	"k8s.io/api/admission/v1beta1"
	whv1beta1 "k8s.io/api/admissionregistration/v1beta1"
)

// The AdmitFunc returns response.
type AdmitFunc func(v1beta1.AdmissionReview) error

type AdmissionService struct {
	Path    string
	Func    AdmitFunc
	Handler AdmissionHandler

	ValidatingConfig *whv1beta1.ValidatingWebhookConfiguration
	MutatingConfig   *whv1beta1.MutatingWebhookConfiguration
}
