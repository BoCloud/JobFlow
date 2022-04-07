package router

import (
	"k8s.io/api/admission/v1beta1"
	whv1 "k8s.io/api/admissionregistration/v1"
)

// The AdmitFunc returns response.
type AdmitFunc func(v1beta1.AdmissionReview) error

type AdmissionService struct {
	Path    string
	Func    AdmitFunc
	Handler AdmissionHandler

	ValidatingConfig *whv1.ValidatingWebhookConfiguration
	MutatingConfig   *whv1.MutatingWebhookConfiguration
}
