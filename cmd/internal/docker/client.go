package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

type Client struct {
	cli *client.Client
}

func NewClientWithHost(host string) (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	_, err = cli.Ping(context.Background(), client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker at %s: %w", host, err)
	}
	return &Client{cli: cli}, nil
}
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	_, err = cli.Ping(context.Background(), client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker daemon: %w", err)
	}

	return &Client{cli: cli}, nil
}
