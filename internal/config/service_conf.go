package config

import (
	bankingsdk "github.com/aeolus3000/lendo-sdk/banking"
	"github.com/aeolus3000/lendo-sdk/messaging"
	"time"
)

type ServiceConf struct {
	PollerConf PollerConf
	EsConf ExecutorServiceConf
	PubConf messaging.RabbitMqConfiguration
	SubConf messaging.RabbitMqConfiguration
	BankingConf bankingsdk.Configuration
}

type ExecutorServiceConf struct {
	Workers int `default:"2"`
	QueueLength int `default:"50"`
}

type PollerConf struct {
	PollInterval time.Duration `default:"2s"`
	WorkloadTimeout time.Duration `default:"25s"`
	RetryInterval time.Duration `default:"2s"`
}