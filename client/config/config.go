package config

import (
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	DefaultPool string `yaml:"default_pool"`
}

func ReadConfiguration(filePath string) (*Configuration, error) {
	config := &Configuration{
		Host: "http://localhost",
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
	defer func(file *os.File) {
		_ = file.Close()

	}(file)

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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	_, err = file.Write(data)

	return err
}
