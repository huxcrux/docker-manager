package config

import (
	"github.com/docker/go-connections/nat"
	docker "github.com/huxcrux/docker-manager/pkg/docker"
)

var containers []docker.ContainerConfig

func ConfigToDockerConfig(config Config) ([]docker.ContainerConfig, error) {
	for container := range config.Containers {

		// generate portset
		portSet := make(nat.PortSet)

		for _, portBinding := range config.Containers[container].PortBindings {
			port, err := nat.NewPort(portBinding.Protocol, portBinding.Port)
			if err != nil {
				return nil, err
			}
			portSet[port] = struct{}{}
		}

		// generate portmap
		portMap := make(nat.PortMap)
		for _, portBinding := range config.Containers[container].PortBindings {
			port, err := nat.NewPort(portBinding.Protocol, portBinding.Port)
			if err != nil {
				return nil, err
			}
			portMap[port] = []nat.PortBinding{
				{
					HostIP:   portBinding.HostIP,
					HostPort: portBinding.HostPort,
				},
			}
		}

		localContainer := docker.ContainerConfig{
			Image:        config.Containers[container].Image,
			Name:         config.Containers[container].Name,
			ExposedPorts: portSet,
			PortBindings: portMap,
			Env:          config.Containers[container].Env,
			Cmd:          config.Containers[container].Cmd,
		}
		containers = append(containers, localContainer)
	}

	return containers, nil
}
