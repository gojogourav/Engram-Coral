package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Command struct {
	Action    string
	Namespace string
	Target    string
	Replicas  int32
	TailLines int64
}

func InterpretCommand(translatedText string) (Command, error) {
	text := strings.ToLower(strings.TrimSpace(translatedText))
	cmd := Command{Namespace: "", TailLines: 50}

	switch {
	case contains(text, "crashing", "crash", "failed pods"):
		cmd.Action = "get_crashing_pods"
	case contains(text, "all pods", "list pods", "show pods", "status"):
		cmd.Action = "get_pods"
	case contains(text, "log", "logs"):
		cmd.Action = "get_logs"
		cmd.Target = extractTarget(text)
	case contains(text, "scale"):
		cmd.Action = "scale"
		cmd.Target = extractTarget(text)
		cmd.Replicas = extractReplicas(text)
	case contains(text, "restart", "rollback"):
		cmd.Action = "restart"
		cmd.Target = extractTarget(text)
	case contains(text, "deployment", "deployments"):
		cmd.Action = "get_deployments"
	default:
		return cmd, fmt.Errorf("could not interpret k8s command: %q", translatedText)
	}

	return cmd, nil
}

func (c *Client) ExecuteCommand(cmd Command) (string, error) {
	ctx := context.Background()

	// If namespace is empty and we have a target, find which namespace it's in
	if cmd.Namespace == "" && cmd.Target != "" {
		ns, err := c.findDeploymentNamespace(ctx, cmd.Target)
		if err == nil {
			cmd.Namespace = ns
		}
	}

	switch cmd.Action {
	case "scale":
		if cmd.Target == "" {
			return "", fmt.Errorf("no deployment name specified")
		}
		if cmd.Replicas == 0 {
			return "", fmt.Errorf("could not determine replica count")
		}
		scale, err := c.kube.AppsV1().Deployments(cmd.Namespace).GetScale(
			ctx, cmd.Target, metav1.GetOptions{},
		)
		if err != nil {
			return "", fmt.Errorf("failed to get scale: %w", err)
		}
		scale.Spec.Replicas = cmd.Replicas
		_, err = c.kube.AppsV1().Deployments(cmd.Namespace).UpdateScale(
			ctx, cmd.Target, scale, metav1.UpdateOptions{},
		)
		if err != nil {
			return "", fmt.Errorf("failed to scale: %w", err)
		}
		return fmt.Sprintf("Scaled %s to %d replicas", cmd.Target, cmd.Replicas), nil

	case "get_pods":
		pods, err := c.kube.CoreV1().Pods(cmd.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list pods: %w", err)
		}
		if len(pods.Items) == 0 {
			return "No pods found.", nil
		}
		var sb strings.Builder
		for _, p := range pods.Items {
			sb.WriteString(fmt.Sprintf("Pod: %s | Status: %s | Namespace: %s\n",
				p.Name, string(p.Status.Phase), p.Namespace))
		}
		return sb.String(), nil

	case "get_crashing_pods":
		pods, err := c.kube.CoreV1().Pods(cmd.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list pods: %w", err)
		}
		var sb strings.Builder
		found := false
		for _, p := range pods.Items {
			var restarts int32
			for _, cs := range p.Status.ContainerStatuses {
				restarts += cs.RestartCount
			}
			if string(p.Status.Phase) == "Failed" || restarts > 3 {
				sb.WriteString(fmt.Sprintf("Pod: %s | Status: %s | Restarts: %d\n",
					p.Name, string(p.Status.Phase), restarts))
				found = true
			}
		}
		if !found {
			return "No crashing pods found.", nil
		}
		return "Crashing pods:\n" + sb.String(), nil

	case "get_deployments":
		deployments, err := c.kube.AppsV1().Deployments(cmd.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list deployments: %w", err)
		}
		var sb strings.Builder
		for _, d := range deployments.Items {
			sb.WriteString(fmt.Sprintf("Deployment: %s | Ready: %d/%d\n",
				d.Name, d.Status.ReadyReplicas, *d.Spec.Replicas))
		}
		return sb.String(), nil

	case "get_logs":
		if cmd.Target == "" {
			return "", fmt.Errorf("no pod or deployment name specified for logs")
		}
		// Find the first running pod for this deployment
		pods, err := c.kube.CoreV1().Pods(cmd.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", cmd.Target),
		})
		if err != nil {
			return "", fmt.Errorf("failed to list pods for %s: %w", cmd.Target, err)
		}
		if len(pods.Items) == 0 {
			return fmt.Sprintf("No pods found for %s", cmd.Target), nil
		}
		podName := pods.Items[0].Name
		tailLines := cmd.TailLines
		req := c.kube.CoreV1().Pods(cmd.Namespace).GetLogs(podName, &corev1.PodLogOptions{
			TailLines: &tailLines,
		})
		logs, err := req.Stream(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to stream logs for pod %s: %w", podName, err)
		}
		defer logs.Close()
		buf := new(strings.Builder)
		_, err = fmt.Fscan(logs, buf)
		if err != nil && err.Error() != "EOF" {
			return "", fmt.Errorf("failed to read logs: %w", err)
		}
		return fmt.Sprintf("Logs for pod %s:\n%s", podName, buf.String()), nil

	case "restart":
		if cmd.Target == "" {
			return "", fmt.Errorf("no deployment name specified")
		}
		d, err := c.kube.AppsV1().Deployments(cmd.Namespace).Get(ctx, cmd.Target, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get deployment: %w", err)
		}
		if d.Spec.Template.Annotations == nil {
			d.Spec.Template.Annotations = make(map[string]string)
		}
		d.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().String()
		_, err = c.kube.AppsV1().Deployments(cmd.Namespace).Update(ctx, d, metav1.UpdateOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to restart: %w", err)
		}
		return fmt.Sprintf(" Restarted deployment %s", cmd.Target), nil

	default:
		return "", fmt.Errorf("unknown action: %s", cmd.Action)
	}
}

func (c *Client) GetDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error) {
	deployments, err := c.kube.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	var result []DeploymentInfo
	for _, d := range deployments.Items {
		result = append(result, DeploymentInfo{
			Name:      d.Name,
			Namespace: d.Namespace,
			Ready:     d.Status.ReadyReplicas,
			Desired:   *d.Spec.Replicas,
		})
	}
	return result, nil
}

func (c *Client) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	scale, err := c.kube.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get scale: %w", err)
	}
	scale.Spec.Replicas = replicas
	_, err = c.kube.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale deployment: %w", err)
	}
	return nil
}

func (c *Client) RestartDeployment(ctx context.Context, namespace, name string) error {
	d, err := c.kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}
	if d.Spec.Template.Annotations == nil {
		d.Spec.Template.Annotations = make(map[string]string)
	}
	d.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().String()
	_, err = c.kube.AppsV1().Deployments(namespace).Update(ctx, d, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to restart deployment: %w", err)
	}
	return nil
}

// DeploymentInfo is a serializable summary of a deployment
type DeploymentInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Ready     int32  `json:"ready"`
	Desired   int32  `json:"desired"`
}

func (c *Client) findDeploymentNamespace(ctx context.Context, name string) (string, error) {
	namespaces := []string{"default", "voxdeploy", "monitoring", "kube-system"}
	for _, ns := range namespaces {
		_, err := c.kube.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return ns, nil
		}
	}
	return "", fmt.Errorf("deployment %s not found in any namespace", name)
}

func contains(text string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func extractTarget(text string) string {
	words := strings.Fields(text)
	skipWords := map[string]bool{
		"deployment": true,
		"pod":        true,
		"container":  true,
		"the":        true,
	}
	triggers := []string{"restart", "scale", "logs", "stop", "start"}
	for i, word := range words {
		for _, t := range triggers {
			if word == t {
				for j := i + 1; j < len(words); j++ {
					if !skipWords[words[j]] {
						return words[j]
					}
				}
			}
		}
	}
	return ""
}

func extractReplicas(text string) int32 {
	words := strings.Fields(text)
	for _, word := range words {
		var n int32
		if _, err := fmt.Sscanf(word, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return 0
}
