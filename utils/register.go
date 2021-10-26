package utils

import (
	"errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path"
	batchv1alpha2 "volcano.sh/apis/pkg/client/clientset/versioned/typed/batch/v1alpha1"
)

func init() {
	VcClient, _ = GetVcclient()
}

var VcClient *batchv1alpha2.BatchV1alpha1Client

func GetVcclient() (*batchv1alpha2.BatchV1alpha1Client, error) {
	config, err := GetKubeConfig()
	if err != nil {
		panic(err.Error())
	}
	vcclient := batchv1alpha2.NewForConfigOrDie(config)
	if err != nil {
		return nil, err
	}
	return vcclient, nil
}

func GetKubeclient() (*kubernetes.Clientset, error) {
	config, err := GetKubeConfig()
	if err != nil {
		panic(err.Error())
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func GetKubeConfig() (*rest.Config, error) {
	var kubeconfig string
	if home := homeDir(); home != "" {
		kubeconfig = path.Join(home, ".kube", "config")
	} else {
		return nil, errors.New("init vcvlient config error")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
