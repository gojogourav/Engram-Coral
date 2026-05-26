package docker

import (
	"fmt"
	"strings"
)

type Command struct {
	Action    string
	Target    string
	TailLines string
}

func InterpretCommand(translatedText string) (Command, error) {
	text := strings.ToLower(strings.TrimSpace(translatedText))
	cmd := Command{TailLines: "50"}

	switch {
	case contains(text, "list", "show", "all containers", "running containers"):
		cmd.Action = "list"
	case contains(text, "stop"):
		cmd.Action = "stop"
		cmd.Target = extractTarget(text)
	case contains(text, "start"):
		cmd.Action = "start"
		cmd.Target = extractTarget(text)
	case contains(text, "restart"):
		cmd.Action = "restart"
		cmd.Target = extractTarget(text)
	case contains(text, "log", "logs"):
		cmd.Action = "logs"
		cmd.Target = extractTarget(text)
	default:
		return cmd, fmt.Errorf("could not interpret docker command: %q", translatedText)
	}

	return cmd, nil
}

func (c *Client) ExecuteCommand(cmd Command) (string, error) {

	// If namespace is empty and we have a target, find which namespace it's in

	switch cmd.Action {
	case "list":
		containers, err := c.GetContainers(false)
		if err != nil {
			return "", err
		}
		return FormatContainers(containers), nil
	case "stop":
		if cmd.Target == "" {
			return "", fmt.Errorf("no container name specified")
		}
		if err := c.StopContainer(cmd.Target); err != nil {
			return "", err
		}
		return fmt.Sprintf(" Stopped container %s", cmd.Target), nil
	case "start":
		if cmd.Target == "" {
			return "", fmt.Errorf("no container name specified")
		}
		if err := c.StartContainer(cmd.Target); err != nil {
			return "", err
		}
		return fmt.Sprintf("Started container %s", cmd.Target), nil
	case "restart":
		if cmd.Target == "" {
			return "", fmt.Errorf("no container name specified")
		}
		if err := c.RestartContainer(cmd.Target); err != nil {
			return "", err
		}
		return fmt.Sprintf("Restarted container %s", cmd.Target), nil
	case "logs":
		if cmd.Target == "" {
			return "", fmt.Errorf("no container name specified")
		}
		return c.GetContainerLogs(cmd.Target, cmd.TailLines)
	default:
		return "", fmt.Errorf("unknown action: %s", cmd.Action)
	}
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
	triggers := []string{"container", "stop", "start", "restart", "logs"}
	for i, word := range words {
		for _, t := range triggers {
			if word == t && i+1 < len(words) {
				return words[i+1]
			}
		}
	}
	return ""
}
