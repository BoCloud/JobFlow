package utils

import (
	"errors"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path"
	batchv1alpha2 "volcano.sh/apis/pkg/client/clientset/versioned/typed/batch/v1alpha1"
)

var (
	client                    *kubernetes.Clientset
	podLister                 corelisters.PodLister
	myPodNamenaspace          string
	denpendentTaskMinReplicas *int32
	denpendentTaskName        string
)

func GetVcclient() (*batchv1alpha2.BatchV1alpha1Client, error) {
	var kubeconfig string
	if home := homeDir(); home != "" {
		kubeconfig = path.Join(home, ".kube", "config")
	} else {
		return nil, errors.New("init vcvlient config error")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	//config, err := rest.InClusterConfig()
	//if err != nil {
	//	return nil, err
	//}
	vcclient := batchv1alpha2.NewForConfigOrDie(config)
	if err != nil {
		return nil, err
	}
	return vcclient, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

//func registryClient() (*batchv1alpha2.BatchV1alpha1Client,error) {
//	config, err := rest.InClusterConfig()
//	if err != nil {
//		return err
//	}
//
//	client, err = kubernetes.NewForConfig(config)
//	if err != nil {
//		return err
//	}
//
//	vcclient = batchv1alpha2.NewForConfigOrDie(config)
//	if err != nil {
//		return err
//	}
//
//	vcclient.Jobs()
//
//	informer := informers.NewSharedInformerFactory(client, InformerSyncPeriod).Core().V1().Pods()
//	podLister = informer.Lister()
//	podSynced := informer.Informer().HasSynced
//
//	var stopCh <-chan struct{}
//	go informer.Informer().Run(stopCh)
//	if ok := cache.WaitForCacheSync(stopCh, podSynced); !ok {
//		return fmt.Errorf("failed to wait for caches to sync")
//	}
//	klog.Infoln("Informer cache was synced successfully.")
//	return nil
//}
