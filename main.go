package main

import (
	"github.com/aeolus3000/lendo-polling-service/internal"
	"github.com/aeolus3000/lendo-sdk/application"
	"os"
)

// Main function
func main() {
	ps := internal.PollingService{}
	application.NewBootstrapApplication(&ps, internal.ApplicationName).Execute(os.Args)
}
