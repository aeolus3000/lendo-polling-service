package service

import (
	"github.com/aeolus3000/lendo-polling-service/internal/config"
	"github.com/aeolus3000/lendo-polling-service/internal/job"
	bankingsdk "github.com/aeolus3000/lendo-sdk/banking"
	"github.com/aeolus3000/lendo-sdk/banking/dnb"
	"github.com/aeolus3000/lendo-sdk/executor"
	"github.com/aeolus3000/lendo-sdk/messaging"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

type Poller struct {
	es              *executor.ExecutorService
	bank            bankingsdk.BankingApi
	publisher       messaging.Publisher
	subscriber      messaging.Subscriber
	workloadTimeout time.Duration
	retryInterval   time.Duration
}

func NewPoller(config config.ServiceConf, done <-chan os.Signal) *Poller {

	es := executor.NewExecutorService(config.EsConf.Workers, config.EsConf.QueueLength)
	pub, err := messaging.NewRabbitMqPublisher(config.PubConf, done)
	if err != nil {
		log.Panicf("NewPoller: Can't start publisher; error = %v", err)
	}
	sub, err := messaging.NewRabbitMqSubscriber(config.SubConf, done)
	if err != nil {
		log.Panicf("NewPoller: Can't start subscriber; error = %v", err)
	}
	return &Poller{
		es:              &es,
		bank:            dnb.NewDnbBanking(config.BankingConf),
		publisher:       pub,
		subscriber:      sub,
		workloadTimeout: config.PollerConf.WorkloadTimeout,
		retryInterval:   config.PollerConf.RetryInterval,
	}
}

func (p *Poller) Poll() {
	msgs, err := p.subscriber.Consume()
	if err != nil {
		log.Errorf("Poll: Not able to consume messages; error: %v", err)
	}
	for msg := range msgs {
		log.Info("Received job")
		workload := job.Workload{
			Pub:             p.publisher,
			Bank:            p.bank,
			Message:         msg,
			WorkloadTimeout: p.workloadTimeout,
			RetryInterval:   p.retryInterval,
		}
		p.es.QueueJob(&workload)
	}
}

func (p *Poller) Shutdown() {
	errSub := p.subscriber.Close()
	if errSub != nil {
		log.Errorf("Failed to shutdown sub; error = %v", errSub)
	}
	errPub := p.publisher.Close()
	if errPub != nil {
		log.Errorf("Failed to shutdown sub; error = %v", errPub)
	}
}
