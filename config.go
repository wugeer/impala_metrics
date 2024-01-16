package main

import (
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

type Config struct {
	Servers         []string `yaml:"servers"`
	IntervalSeconds int      `yaml:"intervalSeconds"`
	Port            int      `yaml:"port"`
	NumWorkers      int      `yaml:"numWorkers"`
	Metrics         []string `yaml:"metrics"`
}

func loadConfig(configFilePath string, logger *log.Logger) (*Config, error) {
	yamlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		logger.Fatalf("读取文件时发生错误: %v", err)
	}
	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
