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
	cli, err := client.New(
		client.WithHost(host),
		// client.WithAPIVersionNegotiation()
	)

	if err != nil {
		return nil, fmt.Errorf("Failed to createe docker isntance - %W", err)
	}

	_, err = cli.Ping(context.Background(), client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to docker - %w", err)
	}

	return &Client{cli: cli}, nil
}

func NewClient() (*Client, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("Failed to create docker client, %w", err)
	}

	_, err = cli.Ping(context.Background(), client.PingOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to docker - %w", err)
	}

	return &Client{cli: cli}, nil
}
