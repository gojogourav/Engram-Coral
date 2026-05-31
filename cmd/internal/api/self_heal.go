package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type RemediationAction struct {
	Action   string `json:"action"`   // "scale" or "restart"
	Target   string `json:"target"`   // Name of the deployment
	Replicas int32  `json:"replicas"` // Only used if action is "scale"
}

func (g *Gateway) TriggerSelfHealing(alertName, podName, namespace, alertDescription string) {
	log.Printf(" [RUNBOOK TRIPPED] Alert: %s | Pod: %s", alertName, podName)

	// var sqlQuery string

	// switch alertName {
	// case "OOMKilled", "HighMemoryUsage":
	// 	log.Printf("Running memory investigation runbook...")
	// 	sqlQuery = fmt.Sprintf(`SELECT name, status, restarts, memory_usage, memory_limit
	// 		FROM k8s.pods
	// 		JOIN metrics.pods ON name = pod_name
	// 		WHERE name = '%s' AND namespace = '%s'`, podName, namespace)

	// case "CrashLoopBackOff", "PodCrashLooping":
	// 	log.Printf(" Running crash investigation runbook...")
	// 	sqlQuery = fmt.Sprintf(`SELECT p.name, p.status, p.restarts, e.message
	// 		FROM k8s.pods p
	// 		JOIN k8s.events e ON p.name = e.pod_name
	// 		WHERE p.name = '%s' AND p.namespace = '%s'
	// 		ORDER BY e.time DESC LIMIT 5`, podName, namespace)

	// case "HighCPU", "CPUSaturation":
	// 	log.Printf("🔍 Running CPU saturation runbook...")
	// 	sqlQuery = fmt.Sprintf(`SELECT name, cpu_usage, cpu_limit
	// 		FROM k8s.pods
	// 		JOIN metrics.pods ON name = pod_name
	// 		WHERE name = '%s' AND namespace = '%s'`, podName, namespace)

	// default:
	// 	log.Printf(" Unknown alert '%s', falling back to general pod inspection...", alertName)
	// 	sqlQuery = fmt.Sprintf(`SELECT * FROM k8s.pods WHERE name = '%s' AND namespace = '%s'`, podName, namespace)
	// }

	// cmd := exec.Command("coral", "sql", sqlQuery, "--format", "json")
	// log.Printf(" Bypassing Coral. Fetching pod state natively via kubectl...")

	// cmd := exec.Command("kubectl", "get", "pod", podName, "-n", namespace, "-o", "json")
	// clusterData, err := cmd.CombinedOutput()
	// if err != nil {
	// 	log.Printf(" kubectl execution failed: %v\n Log:\n%s", err, string(clusterData))
	// 	return
	// }
	// log.Printf("Retrieved pod JSON state successfully.")

	log.Printf(" Executing diagnostic trace via Coral SQL fabric...")

	// 1. Fetch the exact Pod State natively through Coral
	podQuery := fmt.Sprintf("SELECT * FROM k8s.pods WHERE name = '%s' AND namespace = '%s' LIMIT 1", podName, namespace)
	podData, err := runCoralQuery(podQuery)
	if err != nil {
		log.Printf("❌ Coral pod query failed: %v", err)
		return
	}

	// 2. Fetch recent Warning events in the namespace so the AI knows WHY it crashed
	eventQuery := fmt.Sprintf("SELECT reason, message, count FROM k8s.events WHERE namespace = '%s' AND type = 'Warning' LIMIT 3", namespace)
	eventData, _ := runCoralQuery(eventQuery) // We ignore errors here so the pipeline continues even if there are no events

	// Combine them into a clean text block for the LLM
	clusterData := fmt.Sprintf("=== POD STATE ===\n%s\n\n=== RECENT WARNINGS ===\n%s", podData, eventData)

	// clusterData, err := cmd.CombinedOutput()
	// if err != nil {
	// 	log.Printf(" Coral SQL execution failed: %v \n Coral Error Log:%s", err, string(clusterData))
	// 	return
	// }
	log.Printf(" Coral retrieved cluster state in milliseconds.")

	remediationPrompt := fmt.Sprintf(`
You are an SRE AI. The pod '%s' threw a '%s' alert.
Here is the live diagnostic data retrieved via Coral SQL:

%s

Look at the "labels" in the POD STATE to determine the exact deployment name (usually the "app" label).
Decide how to fix it based on the RECENT WARNINGS and status.
Respond ONLY with a raw JSON object matching this schema, no markdown:
{
  "action": "restart" or "scale",
  "target": "name_of_the_deployment",
  "replicas": <int> (if scaling, add 1 to current desired replicas. If restarting, set to 0)
}`, podName, alertName, clusterData)
	remediationDecision, err := g.LLMClient.Generate(remediationPrompt)
	if err != nil {
		log.Printf("❌ AI failed to decide on a fix: %v", err)
		return
	}

	fmt.Println(remediationDecision)

	var action RemediationAction
	if err := json.Unmarshal([]byte(strings.TrimSpace(remediationDecision)), &action); err != nil {
		log.Printf(" Failed to parse AI remediation JSON: %v", err)
		return
	}

	ctx := context.Background()
	log.Printf(" AI Executing Fix: Action=%s, Target=%s, Replicas=%d", action.Action, action.Target, action.Replicas)

	switch action.Action {
	case "restart":
		err = g.K8sClient.RestartDeployment(ctx, namespace, action.Target)
	case "scale":
		err = g.K8sClient.ScaleDeployment(ctx, namespace, action.Target, action.Replicas)
	default:
		log.Printf("AI suggested unknown action: %s", action.Action)
		return
	}

	if err != nil {
		log.Printf("Kubernetes execution failed: %v", err)
		return
	}

	log.Printf(" CLUSTER HEALED: Successfully applied %s to %s", action.Action, action.Target)
}
