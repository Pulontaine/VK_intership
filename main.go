package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/Pulontaine/VK_internship/pubsub"
	"google.golang.org/grpc"
)

type Config struct {
	Port string `json:"port"`
}

func loadConfig(filePath string) (Config, error) {
	var config Config
	data, err := os.ReadFile(filePath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %v", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse config file: %v", err)
	}
	if config.Port == "" {
		return config, fmt.Errorf("port is not specified in config")
	}
	return config, nil
}

func main() {
	config, _ := loadConfig("config.json")

	_, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", config.Port, err)
	}

	grpcServer := grpc.NewServer()
	pubsub.RegisterPubSubServer(grpcServer, pubsub.NewServer())

	log.Printf("Starting gRPC server on : %s", config.Port)
}
