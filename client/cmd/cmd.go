package main

import (
	"continuity/client"
	"continuity/client/config"
	"continuity/client/version"
	"log"

	"github.com/spf13/cobra"
)

var configuration *config.Configuration
var c client.Client

func main() {
	var filePath string
	var err error
	rootCmd := &cobra.Command{
		Use:   "continuity-client",
		Short: "continuity client",
	}

	rootCmd.Version = version.Version

	rootCmd.PersistentFlags().StringVarP(&filePath, "file", "f", "", "configuration file path")
	rootCmd.Version = version.Version
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Use != "sample-config" {
			configuration, err = config.ReadConfiguration(filePath)
			if err != nil {
				log.Fatalf("Error reading configuration: %v", err)
			}
			c = client.NewClient(configuration)
			v, err := c.GetVersion()
			if err != nil {
				log.Fatalf("Error getting server version: %v", err)
			}
			if v != version.Version {
				log.Fatalf("Error getting server version. Expected %s, got %s", version.Version, v)
			}
		}
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "sample-config",
		Short: "Generate a sample configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			sampleConfig := &config.Configuration{
				Host: "localhost",
				Port: 8090,
			}
			err := config.WriteSampleConfiguration(sampleConfig)
			if err != nil {
				log.Fatalf("Error generating sample configuration: %v", err)
			}
			log.Println("Sample configuration file generated: config.yaml")
		},
	})

	rootCmd.AddCommand(poolCmd)
	rootCmd.AddCommand(serverCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
