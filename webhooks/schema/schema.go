package schema

import (
	"fmt"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog"
	corev1 "k8s.io/kubernetes/pkg/apis/core/v1"

	jobflowv1alpha1 "jobflow/api/v1alpha1"
)

func init() {
	addToScheme(scheme)
}

var scheme = runtime.NewScheme()

// Codecs is for retrieving serializers for the supported wire formats
// and conversion wrappers to define preferred internal and external versions.
var Codecs = serializer.NewCodecFactory(scheme)

func addToScheme(scheme *runtime.Scheme) {
	corev1.AddToScheme(scheme)
	v1beta1.AddToScheme(scheme)
}

// DecodeJob decodes the jobFlow using deserializer from the raw object.
func DecodeJobFlow(object runtime.RawExtension, resource metav1.GroupVersionResource) (*jobflowv1alpha1.JobFlow, error) {
	jobFlowResource := metav1.GroupVersionResource{Group: jobflowv1alpha1.GroupVersion.Group, Version: jobflowv1alpha1.GroupVersion.Version, Resource: "jobflows"}
	raw := object.Raw
	jobFlow := jobflowv1alpha1.JobFlow{}

	if resource != jobFlowResource {
		err := fmt.Errorf("expect resource to be %s", jobFlowResource)
		return &jobFlow, err
	}

	deserializer := Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &jobFlow); err != nil {
		return &jobFlow, err
	}
	klog.V(3).Infof("the jobFlow struct is %+v", jobFlow)

	return &jobFlow, nil
}

// DecodeJob decodes the jobTemplate using deserializer from the raw object.
func DecodeJobTemplate(object runtime.RawExtension, resource metav1.GroupVersionResource) (*jobflowv1alpha1.JobTemplate, error) {
	jobTemplateResource := metav1.GroupVersionResource{Group: jobflowv1alpha1.GroupVersion.Group, Version: jobflowv1alpha1.GroupVersion.Version, Resource: "jobtemplates"}
	raw := object.Raw
	jobTemplate := jobflowv1alpha1.JobTemplate{}

	if resource != jobTemplateResource {
		err := fmt.Errorf("expect resource to be %s", jobTemplateResource)
		return &jobTemplate, err
	}

	deserializer := Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &jobTemplate); err != nil {
		return &jobTemplate, err
	}
	klog.V(3).Infof("the jobTemplate struct is %+v", jobTemplate)

	return &jobTemplate, nil
}
