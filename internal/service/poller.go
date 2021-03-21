package service

import (
	bankingsdk "github.com/aeolus3000/lendo-sdk/banking"
	"github.com/aeolus3000/lendo-sdk/banking/dnb"
	"github.com/aeolus3000/lendo-sdk/executor"
	"github.com/aeolus3000/lendo-sdk/messaging"
	log "github.com/sirupsen/logrus"
	"lendo-polling-service/internal/config"
	"lendo-polling-service/internal/job"
	"os"
)

type Poller struct {
	es *executor.ExecutorService
	bank bankingsdk.BankingApi
	publisher messaging.Publisher
	subscriber messaging.Subscriber
}

func NewPoller(config config.ServiceConf, done <-chan os.Signal) Poller {

	es := executor.NewExecutorService(config.EsConf.Workers, config.EsConf.QueueLength)
	pub, err := messaging.NewRabbitMqPublisher(config.PubConf, done)
	if err != nil {
		log.Panicf("NewPoller: Can't start publisher; error = %v", err)
	}
	sub, err := messaging.NewRabbitMqSubscriber(config.SubConf, done)
	if err != nil {
		log.Panicf("NewPoller: Can't start subscriber; error = %v", err)
	}
	return Poller{
		es:         &es,
		bank: 		dnb.NewDnbBanking(dnb.NewDnbDefaultConfiguration()),
		publisher:  pub,
		subscriber: sub,
	}
}

func (p *Poller) Poll() {
	msgs, err := p.subscriber.Consume()
	if err != nil {
		log.Error("Poll: Not able to consume messages; error: %v", err)
	}
	for msg := range msgs {
		workload := job.Workload{
			Pub:        p.publisher,
			Bank:       p.bank,
			Message:    &msg,
		}
		p.es.QueueJob(&workload)
	}
}

func (p *Poller) Shutdown() {
	errSub := p.subscriber.Close()
	if errSub != nil {
		log.Error("Failed to shutdown sub; error = %v", errSub)
	}
	errPub := p.publisher.Close()
	if errPub != nil {
		log.Error("Failed to shutdown sub; error = %v", errPub)
	}
}
