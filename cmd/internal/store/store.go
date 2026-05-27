package store

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"

	"github.com/gojogourav/engram/cmd/internal/docker"
	"github.com/gojogourav/engram/cmd/internal/grafana"
	"github.com/gojogourav/engram/cmd/internal/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type RepoConfig struct {
	GithubToken   string
	WebhookSecret string
	K8sClient     *k8s.Client
	DockerClient  *docker.Client
	GrafanaClient *grafana.Client
}

type RepoStore struct {
	mu    sync.RWMutex
	repos map[string]*RepoConfig
}

func NewRepoStore() *RepoStore {
	return &RepoStore{repos: make(map[string]*RepoConfig)}
}

func (s *RepoStore) Get(fullName string) (*RepoConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.repos[fullName]
	return cfg, ok
}

func (s *RepoStore) Register(fullName string, cfg *RepoConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos[fullName] = cfg
}

func (s *RepoStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var repos []string
	for name := range s.repos {
		repos = append(repos, name)
	}
	return repos
}

func BuildRepoConfig(
	githubToken, webhookSecret, kubeconfigB64, dockerHost, grafanaURL, grafanaToken string,
	httpClient *http.Client,
) (*RepoConfig, error) {
	cfg := &RepoConfig{
		GithubToken:   githubToken,
		WebhookSecret: webhookSecret,
	}

	if kubeconfigB64 != "" {
		kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigB64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode kubeconfig: %w", err)
		}
		restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
		}
		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s client: %w", err)
		}
		cfg.K8sClient = k8s.NewClientFromClientSet(clientset)
	}

	if dockerHost != "" {
		dockerClient, err := docker.NewClientWithHost(dockerHost)
		if err != nil {
			return nil, fmt.Errorf("failed to create docker client: %w", err)
		}
		cfg.DockerClient = dockerClient
	}

	if grafanaURL != "" {
		cfg.GrafanaClient = grafana.NewClient(grafanaURL, grafanaToken, httpClient)
	}

	return cfg, nil
}
