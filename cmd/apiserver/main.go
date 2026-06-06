package main

import (
	"os"

	"github.com/spf13/cobra"

	"metering-service/cmd/apiserver/app"
	"metering-service/config"
)

var (
	rootCMD = &cobra.Command{
		Short: "metering-service",
	}

	configCMD = &cobra.Command{
		Use:   "config",
		Short: "Show resolved settings",
		Run: func(*cobra.Command, []string) {
			config.Load().Show()
		},
	}

	serverCMD = &cobra.Command{
		Use:   "server",
		Short: "Run the application server",
		Run: func(*cobra.Command, []string) {
			app.Run()
		},
	}
)

func main() {
	rootCMD.AddCommand(configCMD)
	rootCMD.AddCommand(serverCMD)
	if err := rootCMD.Execute(); err != nil {
		os.Exit(1)
	}
}
