package main

import (
	"fmt"
	"os"

	"github.com/containerd/containerd/cmd/ctr/app"
	"github.com/containerd/containerd/cmd/ctr/commands/containers"
	"github.com/containerd/containerd/cmd/ctr/commands/content"
	"github.com/containerd/containerd/cmd/ctr/commands/images"
	"github.com/containerd/containerd/cmd/ctr/commands/run"
	"github.com/containerd/containerd/cmd/ctr/commands/snapshots"
	"github.com/containerd/containerd/cmd/ctr/commands/tasks"
	versionCmd "github.com/containerd/containerd/cmd/ctr/commands/version"
	"github.com/containerd/containerd/pkg/seed"
)

func init() {
	seed.WithTimeAndRand()
}

func main() {
	app := app.New()

	// modify for topic
	app.Name = "kubeConEU2020"
	app.Commands = nil
	app.Description = ""
	app.Usage = `

       _          _  __     _           _____            
      | |        | |/ /    | |         / ____|           
   ___| |_ _ __  | ' /_   _| |__   ___| |     ___  _ __  
  / __| __| '__| |  <| | | | '_ \ / _ \ |    / _ \| '_ \ 
 | (__| |_| |    | . \ |_| | |_) |  __/ |___| (_) | | | |
  \___|\__|_|    |_|\_\__,_|_.__/ \___|\_____\___/|_| |_|

containerd CLI
`

	// add basic command for demo
	app.Commands = append(app.Commands,
		versionCmd.Command,
		containers.Command,
		content.Command,
		images.Command,
		run.Command,
		snapshots.Command,
		tasks.Command,
		imageZstdConvertCommand)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "kubeConEU2020: %s\n", err)
		os.Exit(1)
	}
}
