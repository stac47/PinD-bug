package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
)

var sha256Pattern = regexp.MustCompile("^[0-9a-z]+$")

func findImageIDFromStream(str string) string {
	str = strings.TrimSpace(str)
	if sha256Pattern.MatchString(str) {
		return str
	}
	return ""
}

func parseDockerDaemonJsonMessages(r io.Reader) (string, error) {
	var decoder *json.Decoder
	decoder = json.NewDecoder(r)
	imageID := ""
	for {
		var jsonMessage jsonmessage.JSONMessage
		if err := decoder.Decode(&jsonMessage); err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if err := jsonMessage.Error; err != nil {
			return "", err
		}
		if jsonMessage.Aux != nil {
			var r types.BuildResult
			if err := json.Unmarshal(*jsonMessage.Aux, &r); err != nil {
			} else {
				imageID = r.ID
			}
		}
		// Hack for podman <= 4.0.0
		if imageID == "" {
			imageID = findImageIDFromStream(jsonMessage.Stream)
		}
	}
	return strings.TrimPrefix(imageID, "sha256:"), nil
}

func main() {
	contextTarPath := os.Args[1]
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	dockerBuildContext, err := os.Open(contextTarPath)
	defer dockerBuildContext.Close()
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
	}
	buildResponse, err := cli.ImageBuild(ctx, dockerBuildContext, buildOptions)
	imageId, err := parseDockerDaemonJsonMessages(buildResponse.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer buildResponse.Body.Close()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageId,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}
