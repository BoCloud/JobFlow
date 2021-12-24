package webhooks

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"k8s.io/api/admissionregistration/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"jobflow/webhooks/router"
)

const (
	defaultQPS              = 50.0
	defaultBurst            = 100
	defaultPort             = 9443
	webhookServiceName      = "jobflow-webhook-service"
	webhookServiceNamespace = "kube-system"
	defaultEnabledAdmission = "/jobflows/validate,/jobtemplates/validate"
	caCertFilePath          = "/tmp/k8s-webhook-server/serving-certs/ca.crt"
	tlsCertFilePath         = "/tmp/k8s-webhook-server/serving-certs/tls.crt"
	tlsKeyFilePath          = "/tmp/k8s-webhook-server/serving-certs/tls.key"
)

func Run() error {
	caBundle, err := ioutil.ReadFile(caCertFilePath)
	if err != nil {
		return fmt.Errorf("unable to read cacert file (%s): %v", caCertFilePath, err)
	}
	restConfig, err := restclient.InClusterConfig()
	if err != nil {
		return fmt.Errorf("unable to build k8s config: %v", err)
	}
	restConfig.QPS = defaultQPS
	restConfig.Burst = defaultBurst
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("unable to get k8s client: %v", err)
	}
	admissions := strings.Split(strings.TrimSpace(defaultEnabledAdmission), ",")
	router.ForEachAdmission(admissions, func(service *router.AdmissionService) {
		klog.V(3).Infof("Registered '%s' as webhook.", service.Path)
		http.HandleFunc(service.Path, service.Handler)
		klog.V(3).Infof("Registered configuration for webhook <%s>", service.Path)
		registerWebhookConfig(kubeClient, service, caBundle)
	})

	server := &http.Server{
		Addr:      ":" + strconv.Itoa(defaultPort),
		TLSConfig: configTLS(),
	}
	go func() {
		err = server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			klog.Fatalf("ListenAndServeTLS for admission webhook failed: %v", err)
		}

		klog.Info("Volcano Webhook manager started.")
	}()

	return nil
}

func registerWebhookConfig(kubeClient *kubernetes.Clientset, service *router.AdmissionService, caBundle []byte) {
	clientConfig := v1beta1.WebhookClientConfig{
		CABundle: caBundle,
		Service: &v1beta1.ServiceReference{
			Name:      webhookServiceName,
			Namespace: webhookServiceNamespace,
			Path:      &service.Path,
		},
	}

	if service.MutatingConfig != nil {
		for i := range service.MutatingConfig.Webhooks {
			service.MutatingConfig.Webhooks[i].ClientConfig = clientConfig
		}

		service.MutatingConfig.ObjectMeta.Name = webhookConfigName(service.Path)

		if err := registerMutateWebhook(kubeClient, service.MutatingConfig); err != nil {
			klog.Errorf("Failed to register mutating admission webhook (%s): %v",
				service.Path, err)
		} else {
			fmt.Printf("Registered mutating webhook for path <%s>.", service.Path)
			klog.V(3).Infof("Registered mutating webhook for path <%s>.", service.Path)
		}
	}
	if service.ValidatingConfig != nil {
		for i := range service.ValidatingConfig.Webhooks {
			service.ValidatingConfig.Webhooks[i].ClientConfig = clientConfig
		}

		service.ValidatingConfig.ObjectMeta.Name = webhookConfigName(service.Path)

		if err := registerValidateWebhook(kubeClient, service.ValidatingConfig); err != nil {
			klog.Errorf("Failed to register validating admission webhook (%s): %v",
				service.Path, err)
		} else {
			klog.V(3).Infof("Registered validating webhook for path <%s>.", service.Path)
		}
	}
}

func registerMutateWebhook(clientset *kubernetes.Clientset, hook *v1beta1.MutatingWebhookConfiguration) error {
	client := clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations()
	existing, err := client.Get(context.TODO(), hook.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err == nil && existing != nil {
		klog.V(4).Infof("Updating MutatingWebhookConfiguration %v", hook)
		existing.Webhooks = hook.Webhooks
		if _, err := client.Update(context.TODO(), existing, metav1.UpdateOptions{}); err != nil {
			return err
		}
	} else {
		klog.V(4).Infof("Creating MutatingWebhookConfiguration %v", hook)
		if _, err := client.Create(context.TODO(), hook, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func registerValidateWebhook(clientset *kubernetes.Clientset, hook *v1beta1.ValidatingWebhookConfiguration) error {
	client := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()

	existing, err := client.Get(context.TODO(), hook.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err == nil && existing != nil {
		existing.Webhooks = hook.Webhooks
		klog.V(4).Infof("Updating ValidatingWebhookConfiguration %v", hook)
		if _, err := client.Update(context.TODO(), existing, metav1.UpdateOptions{}); err != nil {
			return err
		}
	} else {
		klog.V(4).Infof("Creating ValidatingWebhookConfiguration %v", hook)
		if _, err := client.Create(context.TODO(), hook, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func webhookConfigName(path string) string {
	name := "webhook"
	re := regexp.MustCompile(`-+`)
	raw := strings.Join([]string{name, strings.ReplaceAll(path, "/", "-")}, "-")
	return re.ReplaceAllString(raw, "-")
}

func configTLS() *tls.Config {
	sCert, err := tls.LoadX509KeyPair(tlsCertFilePath, tlsKeyFilePath)
	if err != nil {
		klog.Fatal(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
}
