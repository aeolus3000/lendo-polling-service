package main

import (
	"github.com/aeolus3000/lendo-polling-service/internal/service"
	"github.com/aeolus3000/lendo-sdk/application"
	"os"
)

// Main function
func main() {
	ps := service.PollingService{}
	application.NewBootstrapApplication(&ps, service.ApplicationName).Execute(os.Args)
}
