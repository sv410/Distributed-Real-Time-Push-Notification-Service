// Package config provides configuration management for the notification service
package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig `yaml:"server"`
	Kafka  KafkaConfig  `yaml:"kafka"`
	Redis  RedisConfig  `yaml:"redis"`
	Log    LogConfig    `yaml:"log"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

// KafkaConfig represents Kafka configuration
type KafkaConfig struct {
	BootstrapServers string `yaml:"bootstrap_servers"`
	Topic           string `yaml:"topic"`
	GroupID         string `yaml:"group_id"`
	AutoOffsetReset string `yaml:"auto_offset_reset"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Load loads configuration from YAML file
func Load(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDefaultConfig returns default configuration
func GetDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: "8080",
		},
		Kafka: KafkaConfig{
			BootstrapServers: "localhost:9092",
			Topic:           "push-notifications",
			GroupID:         "notification-consumer-group",
			AutoOffsetReset: "earliest",
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		Log: LogConfig{
			Level: "info",
			File:  "",
		},
	}
}