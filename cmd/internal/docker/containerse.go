package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/moby/moby/client"
)

type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
	Ports  string
}

func (c *Client) GetContainers(onlyRunning bool) ([]ContainerInfo, error) {
	ctx := context.Background()
	result, err := c.cli.ContainerList(ctx, client.ContainerListOptions{All: !onlyRunning})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []ContainerInfo
	for _, ct := range result.Items {
		var ports []string
		for _, p := range ct.Ports {
			if p.PublicPort != 0 {
				ports = append(ports, fmt.Sprintf("%d->%d", p.PublicPort, p.PrivatePort))
			}
		}
		portsStr := strings.Join(ports, ", ")
		if portsStr == "" {
			portsStr = "none"
		}
		name := ct.Names[0]
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		containers = append(containers, ContainerInfo{
			ID:     ct.ID[:12],
			Name:   name,
			Image:  ct.Image,
			Status: ct.Status,
			State:  string(ct.State),
			Ports:  portsStr,
		})
	}
	return containers, nil
}

func (c *Client) StopContainer(nameOrID string) error {
	_, err := c.cli.ContainerStop(context.Background(), nameOrID, client.ContainerStopOptions{})
	return err
}

func (c *Client) StartContainer(nameOrID string) error {
	_, err := c.cli.ContainerStart(context.Background(), nameOrID, client.ContainerStartOptions{})
	return err
}

func (c *Client) RestartContainer(nameOrID string) error {
	_, err := c.cli.ContainerRestart(context.Background(), nameOrID, client.ContainerRestartOptions{})
	return err
}

func (c *Client) GetContainerLogs(nameOrID string, tailLines string) (string, error) {
	result, err := c.cli.ContainerLogs(context.Background(), nameOrID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tailLines,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	content, err := io.ReadAll(result)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}
	return string(content), nil
}

func FormatContainers(containers []ContainerInfo) string {
	if len(containers) == 0 {
		return "No containers found."
	}
	var sb strings.Builder
	for _, ct := range containers {
		sb.WriteString(fmt.Sprintf(
			"Container: %s | Image: %s | State: %s | Status: %s | Ports: %s\n",
			ct.Name, ct.Image, ct.State, ct.Status, ct.Ports,
		))
	}
	return sb.String()
}
