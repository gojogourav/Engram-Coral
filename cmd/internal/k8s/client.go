package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	kube *kubernetes.Clientset
}

func NewClientFromClientSet(clientset *kubernetes.Clientset) *Client {
	return &Client{kube: clientset}
}

func NewClient() (*Client, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get k8s config: %w", err)
		}

	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{kube: clientset}, nil

}

func NewClientFromClientset(clientset *kubernetes.Clientset) *Client {
	return &Client{kube: clientset}
}
