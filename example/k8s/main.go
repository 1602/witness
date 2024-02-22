package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/1602/witness"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"

	"k8s.io/client-go/kubernetes"
)

func main() {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	config.AuthProvider = nil
	config.ExecProvider = &api.ExecConfig{
		Command:            "gke-gcloud-auth-plugin",
		APIVersion:         "client.authentication.k8s.io/v1beta1",
		InstallHint:        "Requires gke-gcloud-auth-plugin",
		ProvideClusterInfo: true,
		InteractiveMode:    api.IfAvailableExecInteractiveMode,
	}

	cl, err := rest.HTTPClientFor(config)
	witness.DebugClient(cl, context.TODO())

	// create the clientset
	clientset, err := kubernetes.NewForConfigAndClient(config, cl)
	if err != nil {
		panic(err.Error())
	}

	for {
		pods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		// Examples for error handling:
		// - Use helper functions like e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		namespace := "kube-system"
		pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "k8s-app=kube-dns"})

		time.Sleep(1 * time.Second)
	}
}
