package main

import (
	"continuity/server/conf"
	"flag"
	"log"
)

func main() {
	configFilePath := flag.String("config", "config.yaml", "Path to configuration file")
	sampleConfig := flag.Bool("sample-config", false, "Generate a sample configuration file")
	flag.Parse()

	if *sampleConfig {
		if err := conf.CreateSampleConfig(*configFilePath); err != nil {
			log.Fatalf("Error generating sample configuration: %v", err)
		}
		return
	}

	lb, api, err := conf.LoadConfig(*configFilePath)

	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)

	}
	log.Println("Started load balancer on", lb.BindAddress, ":", lb.BindPort)
	api.Start()
}
