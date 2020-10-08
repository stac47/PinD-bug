package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{"test-docker-bug"},
		Dockerfile: "Dockerfile",
		// Remove:      true,
		// ForceRemove: true,
	}

	dockerContext, err := archive.TarWithOptions("./data", &archive.TarOptions{})
	if err != nil {
		panic(err)
	}
	_, err = cli.ImageBuild(context.TODO(), dockerContext, buildOptions)
	if err == nil {
		fmt.Println("Should have returned an error")
	} else {
		panic(err)
	}
}
