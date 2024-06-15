package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type ContainerConfig struct {
	Image        string
	Name         string
	ExposedPorts nat.PortSet
	PortBindings nat.PortMap
	Env          []string
	Cmd          []string
}

// deleteContainers deletes multiple Docker containers by their IDs
func DeleteContainer(cli *client.Client, containerId string) error {
	ctx := context.Background()

	if err := cli.ContainerStop(ctx, containerId, container.StopOptions{}); err != nil {
		return err
	}

	if err := cli.ContainerRemove(ctx, containerId, container.RemoveOptions{}); err != nil {
		return err
	}

	return nil
}

func CreateContainer(cli *client.Client, config ContainerConfig) (err error, created bool) {

	ctx := context.Background()

	// get running containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err, false
	}

	// check if container already exists
	if len(containers) > 0 {
		found := false
		for _, container := range containers {
			if container.Names[0] == "/"+config.Name {
				found = true
				break
			}
		}
		if found {
			return nil, false
		}
	}

	_, err = cli.ContainerCreate(ctx, &container.Config{
		Image:        config.Image,
		ExposedPorts: config.ExposedPorts,
		Env:          config.Env,
		Cmd:          config.Cmd,
	}, &container.HostConfig{
		PortBindings: config.PortBindings,
	}, nil, nil, config.Name)
	if err != nil {
		return err, false
	}

	return nil, true
}

func EnsureRunningContainers(cli *client.Client, containerID string) error {
	ctx := context.Background()
	err := cli.ContainerStart(ctx, containerID, container.StartOptions{})
	return err
}

func GetContainerIDByName(cli *client.Client, containerName string) (string, error) {
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", err
	}

	for _, container := range containers {
		if container.Names[0] == "/"+containerName {
			return container.ID, nil
		}
	}

	return "", fmt.Errorf("container %s not found", containerName)
}

func ListAllContariners(cli *client.Client) ([]types.Container, error) {
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	return containers, nil
}
