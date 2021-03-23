package internal

import (
	"github.com/aeolus3000/lendo-polling-service/internal/config"
	"github.com/aeolus3000/lendo-polling-service/internal/service"
	"github.com/aeolus3000/lendo-sdk/application"
	"github.com/aeolus3000/lendo-sdk/configuration"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

const (
	ApplicationName = "pollingservice"
)

type PollingService struct {
	application.AbstractApplication
	cfg            config.ServiceConf
	poller         *service.Poller
	shutdownSignal <-chan os.Signal
}

func (ps *PollingService) Initialize(configuration configuration.Configuration) {
	ps.shutdownSignal = createShutdownSignalReceiver()
	ps.readConfiguration(configuration)
	ps.createPoller()
}

func (ps *PollingService) Execute() {
	go ps.poller.Poll()

	waitForShutdown(ps.shutdownSignal)
}

func (ps *PollingService) Shutdown() {
	ps.poller.Shutdown()
}

func (ps *PollingService) readConfiguration(configuration configuration.Configuration) {
	ps.cfg = config.ServiceConf{}
	err := configuration.Process(ApplicationName, &ps.cfg)
	if err != nil {
		log.Panicf("Can't read configuration: %v", err)
	}
}

func (ps *PollingService) createPoller() {
	poller := service.NewPoller(ps.cfg, ps.shutdownSignal)
	ps.poller = poller
}

func createShutdownSignalReceiver() <-chan os.Signal {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	return signalChannel
}

func waitForShutdown(shutdownSignal <-chan os.Signal) {
	select {
	case <-shutdownSignal:
		log.Infof("waitForShutdown: Shutdown signal received")
	}
}
