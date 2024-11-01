/*
 * Copyright 2024 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Immutos Community Edition License, Version 1.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *    http://immutos.com/licenses/LICENSE-1.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package buildkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	"github.com/immutos/immutos/internal/constants"
	"golang.org/x/term"
)

// StartDaemon starts the BuildKit daemon in a Docker container (if it is not already running).
func (b *BuildKit) StartDaemon(ctx context.Context) error {
	needsRestart, err := refreshCertificates(b.certsDir)
	if err != nil {
		return fmt.Errorf("failed to refresh certificates: %w", err)
	}

	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var containerID string
	for _, c := range containers {
		if c.Names[0] == "/"+b.containerName {
			// Check if the container is already running.
			if c.State == "running" && !needsRestart {
				containerID = c.ID
				goto BUILDKITD_ALREADY_RUNNING
			}

			slog.Debug("Removing existing buildkit container", slog.String("name", b.containerName))

			if err := cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
				return fmt.Errorf("failed to remove existing buildkit container %s: %w", b.containerName, err)
			}
		}
	}

	{
		config := &container.Config{
			Image: constants.BuildKitImage,
			Cmd: []string{
				"--addr", "tcp://0.0.0.0:8443",
				"--tlscert", "/certs/buildkitd.pem",
				"--tlskey", "/certs/buildkitd-key.pem",
				"--tlscacert", "/certs/ca.pem",
			},
			ExposedPorts: map[nat.Port]struct{}{
				"8443/tcp": {},
			},
		}

		hostConfig := &container.HostConfig{
			Privileged: true,
			// Use a random port on the host.
			PortBindings: nat.PortMap{
				nat.Port("8443/tcp"): []nat.PortBinding{
					{
						HostIP:   "127.0.0.1",
						HostPort: "0",
					},
				},
			},
			Mounts: []mount.Mount{
				{
					Type:     mount.TypeBind,
					Source:   b.certsDir,
					Target:   "/certs/",
					ReadOnly: true,
				},
			},
		}

		// Check if the buildkit image is already available.
		_, _, err := cli.ImageInspectWithRaw(ctx, config.Image)
		if err != nil {
			slog.Info("Pulling buildkit image", slog.String("image", config.Image))

			// Pull the buildkit image.
			pullProgressReader, err := cli.ImagePull(ctx, config.Image, types.ImagePullOptions{})
			if err != nil {
				return fmt.Errorf("failed to pull buildkit image: %w", err)
			}
			defer pullProgressReader.Close()

			if err := displayImagePullProgress(ctx, pullProgressReader); err != nil {
				return fmt.Errorf("failed to display buildkit image pull progress: %w", err)
			}
		}

		slog.Debug("Starting buildkit container", slog.String("name", b.containerName))

		resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, b.containerName)
		if err != nil {
			return fmt.Errorf("failed to create buildkit container: %w", err)
		}

		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			return fmt.Errorf("failed to start buildkit container: %w", err)
		}

		containerID = resp.ID
	}

BUILDKITD_ALREADY_RUNNING:

	b.address, err = getBuildKitAddress(ctx, cli, containerID)
	if err != nil {
		return fmt.Errorf("failed to get buildkit address: %w", err)
	}

	if err := waitForBuildKit(ctx, cli, containerID, b.address); err != nil {
		return fmt.Errorf("failed to wait for buildkit container to start: %w", err)
	}

	return nil
}

// StopDaemon stops the BuildKit daemon running in a Docker container.
func (b *BuildKit) StopDaemon(ctx context.Context) error {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		if c.Names[0] == "/"+b.containerName {
			if err := cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
				return fmt.Errorf("failed to remove buildkit container: %w", err)
			}
		}
	}

	return nil
}

func getBuildKitAddress(ctx context.Context, cli *dockerclient.Client, containerID string) (string, error) {
	info, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect buildkit container: %w", err)
	}

	port := info.NetworkSettings.Ports[nat.Port("8443/tcp")]
	if len(port) == 0 {
		return "", fmt.Errorf("failed to get buildkit container port")
	}

	daemonHostURL, err := url.Parse(cli.DaemonHost())
	if err != nil {
		return "", fmt.Errorf("failed to parse daemon host URL: %w", err)
	}

	host := "localhost"
	switch daemonHostURL.Scheme {
	case "http", "https", "tcp":
		host = daemonHostURL.Hostname()
	case "unix", "npipe":
		// Use the default gateway IP (presumably the Docker host) if we are in a container.
		if _, err := os.Stat("/.dockerenv"); err == nil {
			cmd := exec.CommandContext(ctx, "ip", "route")
			stdout, err := cmd.Output()
			if err != nil {
				return "", fmt.Errorf("failed to get default gateway IP: %w", err)
			}

			for _, line := range strings.Split(string(stdout), "\n") {
				if strings.Contains(line, "default") {
					fields := strings.Fields(line)
					host = fields[2]
					break
				}
			}
		}
	default:
		return "", fmt.Errorf("unsupported daemon host scheme: %s" + daemonHostURL.Scheme)
	}

	return "tcp://" + net.JoinHostPort(host, port[0].HostPort), nil
}

func waitForBuildKit(ctx context.Context, cli *dockerclient.Client, containerID, buildkitAddress string) error {
	buildkitURL, err := url.Parse(buildkitAddress)
	if err != nil {
		return fmt.Errorf("failed to parse buildkit address: %w", err)
	}

	if buildkitURL.Scheme != "tcp" {
		return fmt.Errorf("unsupported buildkit address scheme: %s", buildkitURL.Scheme)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Make sure the container is still running.
			info, err := cli.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect buildkit container: %w", err)
			}

			if info.State.Status != "running" {
				return fmt.Errorf("buildkit container is not running")
			}

			// Check if we can connect to the BuildKit daemon.
			conn, err := net.Dial("tcp", buildkitURL.Host)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}

func displayImagePullProgress(ctx context.Context, progressReader io.Reader) error {
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		seen := make(map[string]bool)

		dec := json.NewDecoder(progressReader)
		for {
			var j jsonmessage.JSONMessage
			if err := dec.Decode(&j); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return err
			}

			messageKey := j.ID + j.Status
			if alreadyLogged := seen[messageKey]; !alreadyLogged {
				slog.Debug(j.Status, slog.String("id", j.ID))
				seen[messageKey] = true
			}
		}
	} else {
		if err := jsonmessage.DisplayJSONMessagesStream(progressReader,
			os.Stdout, os.Stdout.Fd(), term.IsTerminal(int(os.Stdout.Fd())), nil); err != nil {
			return err
		}
	}

	return nil
}
