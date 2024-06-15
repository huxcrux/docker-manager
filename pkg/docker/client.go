package docker

import "github.com/docker/docker/client"

// Create client
func CreateClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return cli, nil
}
