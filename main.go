package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/huxcrux/docker-manager/pkg/config"
	"github.com/huxcrux/docker-manager/pkg/docker"
	"github.com/huxcrux/docker-manager/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// Global variable
var (
	cfg   *config.Config
	cfgMu sync.RWMutex
)

func updateConfig() error {
	newcfg, err := config.Read()
	if err != nil {
		return fmt.Errorf("error reading config: %v", err)
	}

	log.Debugf("New config: %+v", newcfg)

	// Use the mutex to prevent race conditions
	cfgMu.Lock()
	cfg = newcfg
	cfgMu.Unlock()

	log.Info("Config reloaded")

	return nil
}

// isContainerUpToDate checks if a running container is using the latest available image
func isContainerUpToDate(cli *client.Client, containerID string, config docker.ContainerConfig) (bool, error) {
	ctx := context.Background()

	// Get the running container's image ID
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	runningImageID := inspect.Image

	// Pull the latest image
	reader, err := cli.ImagePull(ctx, config.Image, image.PullOptions{})
	if err != nil {
		return false, err
	}
	defer reader.Close()
	// Consume the reader to complete the image pull
	_, _ = io.Copy(io.Discard, reader)

	// Get the latest image ID
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return false, err
	}
	var latestImageID string
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == config.Image {
				latestImageID = img.ID
				break
			}
		}
	}

	if latestImageID == "" {
		return false, fmt.Errorf("could not find the latest image for %s", config.Image)
	}

	// Compare the image IDs
	result := runningImageID == latestImageID
	if result {
		log.Debugf("Container %s is up to date\n", config.Name)
	} else {
		log.Debugf("Container %s is not up to date\n", config.Name)
	}

	// Compare the image IDs
	return result, nil
}

// ensureContainerConfig checks if a running container matches the given ContainerConfig and recreates it if necessary
func ensureContainerConfig(cli *client.Client, config docker.ContainerConfig) error {
	ctx := context.Background()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	for _, container := range containers {
		if container.Names[0] == "/"+config.Name {
			inspect, err := cli.ContainerInspect(ctx, container.ID)
			if err != nil {
				return err
			}

			// Validate container configuration
			needsUpdate := false

			// Check environment variables
			// Some env vars is set by container. We need to match the ones we care about. Unclear how we track vars that is unset over time.
			// Skipping for now and will return to this later on.
			//if !reflect.DeepEqual(inspect.Config.Env, config.Env) {
			//	log.Debugf("Container %s environment does not match\n", config.Name)
			//	needsUpdate = true
			//}

			// Check port bindings
			if !reflect.DeepEqual(inspect.Config.ExposedPorts, config.ExposedPorts) {
				log.Debugf("Container %s exposed ports do not match\n", config.Name)
				needsUpdate = true
			}
			if !reflect.DeepEqual(inspect.HostConfig.PortBindings, config.PortBindings) {
				log.Debugf("Container %s port bindings do not match\n", config.Name)
				needsUpdate = true
			}

			// Check image
			if !reflect.DeepEqual(inspect.Config.Image, config.Image) {
				log.Debugf("Container %s image does not match\n", config.Name)
				needsUpdate = true
			}

			// Check command
			if config.Cmd != nil {
				if !reflect.DeepEqual(inspect.Config.Cmd, config.Cmd) {
					log.Debugf("Container %s command does not match\n", config.Name)
					needsUpdate = true
				}
			}

			if needsUpdate {
				log.Infof("Container %s configuration does not match, recreating it...\n", config.Name)

				err = docker.DeleteContainer(cli, container.ID)
				if err != nil {
					return err
				}

				// create container with the correct configuration
				err, created := docker.CreateContainer(cli, config)
				if err != nil {
					return err
				}
				if created {
					log.Infof("Container %s recreated with the correct configuration\n", config.Name)
				}

			} else {
				log.Debugf("Config for container %s already up to date\n", config.Name)
			}
			return nil
		}
	}

	log.Infof("Container %s not found, creating it...\n", config.Name)
	_, err = cli.ContainerCreate(ctx, &container.Config{
		Image:        config.Image,
		ExposedPorts: config.ExposedPorts,
		Env:          config.Env,
		Cmd:          config.Cmd,
	}, &container.HostConfig{
		PortBindings: config.PortBindings,
	}, nil, nil, config.Name)
	if err != nil {
		return err
	}
	return nil
}

// createContainers creates multiple Docker containers based on the provided configurations
func ensureContainers(cli *client.Client, desierdContainers []docker.ContainerConfig, updateCheck bool) error {

	// get running containers
	runningContainers, err := docker.ListAllContariners(cli)
	if err != nil {
		return err
	}

	for _, container := range desierdContainers {
		// check if container already exists
		found := false
		if len(runningContainers) > 0 {
			for _, runningContainer := range runningContainers {
				if runningContainer.Names[0] == "/"+container.Name {
					log.Debugf("Container %s already exists\n", container.Name)
					found = true
					break
				}
			}
		}

		// Create container if not found
		var created bool
		if !found {
			err, created = docker.CreateContainer(cli, container)
			if err != nil {
				return err
			}
			if created {
				log.Infof("Container %s created", container.Name)
			}
		}

		if !created {
			err = ensureContainerConfig(cli, container)
			if err != nil {
				log.Fatalf("Error ensuring container configuration: %v", err)
			}
		}

		// Get cintainer ID from name
		ctid, err := docker.GetContainerIDByName(cli, container.Name)
		if err != nil {
			return err
		}

		// Check if container is up to date
		if updateCheck && !created {
			upToDate, err := isContainerUpToDate(cli, ctid, container)
			if err != nil {
				return err
			}
			if !upToDate {
				log.Infof("Container %v is not up to date, recreating ...\n", container.Name)
				err = docker.DeleteContainer(cli, ctid)

				if err != nil {
					return err
				}

				err, _ := docker.CreateContainer(cli, container)
				if err != nil {
					return err
				}

				// Fetch new container ID
				ctid, err = docker.GetContainerIDByName(cli, container.Name)
				if err != nil {
					return err
				}
			}
		}

		// Ensure container is running
		err = docker.EnsureRunningContainers(cli, ctid)
		if err != nil {
			return err
		}

		log.Infof("Container %v ensured\n", container.Name)
	}

	return nil
}

func removeUnwantedContainers(cli *client.Client, configs []docker.ContainerConfig) error {

	// get running containers
	containers, err := docker.ListAllContariners(cli)
	if err != nil {
		return err
	}

	// check if container is not specified in configs
	for _, container := range containers {
		found := false
		for _, config := range configs {
			if container.Names[0] == "/"+config.Name {
				found = true
				break
			}
		}
		if !found {
			log.Infof("Container %s (%s) not desired, removing ...\n", container.Names[0], container.ID)
			err = docker.DeleteContainer(cli, container.ID)
			if err != nil {
				return err
			}
			log.Debug("Container removed\n")
		}
	}

	return nil
}

// Handler to update metrics and then serve Prometheus metrics
func GenerateMetrics(dm *metrics.DockerMetrics, cli *client.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// List all containers
		containers, err := docker.ListAllContariners(cli)
		if err != nil {
			http.Error(w, "Could not list containers", http.StatusInternalServerError)
			return
		}

		var wg sync.WaitGroup
		statsChan := make(chan types.StatsJSON, len(containers))
		errChan := make(chan error, len(containers))

		// Fetch stats for each container concurrently
		for _, container := range containers {
			wg.Add(1)
			go func(containerID string) {
				defer wg.Done()
				stats, err := cli.ContainerStats(context.Background(), containerID, false)
				//cli.ContainerStatsOneShot(context.Background(), containerID)
				if err != nil {
					errChan <- fmt.Errorf("could not fetch stats for container %s: %v", containerID, err)
					return
				}
				defer stats.Body.Close()

				data, err := io.ReadAll(stats.Body)
				if err != nil {
					errChan <- fmt.Errorf("could not read stats for container %s: %v", containerID, err)
				}

				var statsJSON types.StatsJSON
				err = json.Unmarshal(data, &statsJSON)
				if err != nil {
					errChan <- fmt.Errorf("could not unmarshal stats for container %s: %v", containerID, err)
				}

				log.Infof("Updated metrics for container %s\n", containerID)

				statsChan <- statsJSON
			}(container.ID)
		}

		// Wait for all goroutines to finish
		go func() {
			wg.Wait()
			close(statsChan)
			close(errChan)
		}()

		// Process results
		for statsJSON := range statsChan {
			dm.UpdateMetrics(statsJSON)
		}

		// Handle errors
		if len(errChan) > 0 {
			var errorMsgs []string
			for err := range errChan {
				errorMsgs = append(errorMsgs, err.Error())
			}
			http.Error(w, fmt.Sprintf("Errors occurred: %v", errorMsgs), http.StatusInternalServerError)
			return
		}

		// Serve Prometheus metrics
		promhttp.Handler().ServeHTTP(w, r)
	})
}

func reconcileContainers(cli *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		containers, err := config.ConfigToDockerConfig(*cfg)
		if err != nil {
			log.Fatalf("Error converting config to Docker config: %v", err)
		}

		// Delete unwanted containers
		if cfg.AppConfig.RemoveUnwantedContainers {
			err = removeUnwantedContainers(cli, containers)
			if err != nil {
				log.Fatalf("Error when removing unwanted containers: %v", err)
			}
		}

		// Create containers and ensure they are up to date
		err = ensureContainers(cli, containers, cfg.AppConfig.UpdateCheck)
		if err != nil {
			log.Fatalf("Error ensuring containers: %v", err)
		}

		fmt.Fprint(w, "Containers reconciled\n")
	}
}

func reloadConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updateConfig()
		if err != nil {
			log.Fatalf("Error reloading config: %v", err)
		}
		fmt.Fprint(w, "Config reloaded\n")
	}
}

func init() {
	// read config
	err := updateConfig()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}
}

func main() {
	// if debug is enabled, set log level to debug
	if cfg.AppConfig.Debug {
		log.SetLevel(log.DebugLevel)
	}

	// Create client
	cli, err := docker.CreateClient()
	if err != nil {
		log.Fatalf("Error creating Docker client: %v", err)
	}

	// init metrics
	metrics := metrics.NewDockerMetrics()

	// Expose metrics via HTTP
	http.Handle("/metrics", GenerateMetrics(metrics, cli))
	http.Handle("/update", reconcileContainers(cli))
	http.Handle("/reload", reloadConfig())
	fmt.Println("Beginning to serve on port :8082")
	http.ListenAndServe(":8082", nil)
}
