package main

import (
	"os"

	"github.com/Netflix/titus-executor/vpc/allocate"
	"github.com/Netflix/titus-executor/vpc/context"
	"github.com/Netflix/titus-executor/vpc/gc"
	"github.com/Netflix/titus-executor/vpc/setup"
	"gopkg.in/urfave/cli.v1"
)

// TODO: Add Systemd loggin

func main() {
	app := cli.NewApp()
	app.Name = "titus-vpc-tool"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   context.StateDir,
			Value:  "/run/titus-vpc-tool",
			Usage:  "Where to store the state, and locker state -- creates directory",
			EnvVar: "VPC_STATE_DIR",
		},
		cli.StringFlag{
			Name:  "log-level",
			Value: "info",
		},
	}
	app.Commands = []cli.Command{
		setup.Setup,
		allocate.AllocateNetwork,
		gc.GC,
		allocate.SetupContainer,
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}