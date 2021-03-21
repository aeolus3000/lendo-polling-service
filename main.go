package main

import (
	"github.com/aeolus3000/lendo-sdk/application"
	"lendo-polling-service/internal/service"
)

// Main function
func main() {
	ps := service.PollingService{}
	application.NewBootstrapApplication(&ps, service.ApplicationName)
}