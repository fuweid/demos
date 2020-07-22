package main

import (
	"fmt"
	"os"

	"github.com/containerd/containerd/cmd/ctr/app"
	"github.com/containerd/containerd/pkg/seed"
)

func init() {
	seed.WithTimeAndRand()
}

func main() {
	app := app.New()
	app.Commands = append(app.Commands, imageZstdConvertCommand)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "kubeConEU2020: %s\n", err)
		os.Exit(1)
	}
}
