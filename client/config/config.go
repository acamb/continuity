package config

import (
	"gopkg.in/yaml.v2"
	"io"
	"os"
)

type Configuration struct {
	Host string
	Port int
}

func ReadConfiguration(filePath string) (*Configuration, error) {
	config := &Configuration{
		Host: "localhost",
		Port: 8090,
	}

	if filePath == "" {
		filePath = "config.yaml"
	}

	err := readYaml(filePath, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func readYaml(path string, config *Configuration) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

func WriteSampleConfiguration(config *Configuration) error {
	file, err := os.Create("config.yaml")
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	_, err = file.Write(data)

	return err
}
