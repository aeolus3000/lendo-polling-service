package job

import (
	bankingsdk "github.com/aeolus3000/lendo-sdk/banking"
	"github.com/aeolus3000/lendo-sdk/messaging"
	log "github.com/sirupsen/logrus"
	"time"
)

type Workload struct {
	Pub             messaging.Publisher
	Bank            bankingsdk.BankingApi
	Message         *messaging.Message
	WorkloadTimeout time.Duration
	RetryInterval   time.Duration
}

func (w *Workload) Execute() {
	timeoutChannel := time.After(w.WorkloadTimeout)
	first := time.After(0 * time.Second)
	for {
		application, ok := w.deserializeMessage()
		if !ok {
			w.acknowledgeMessage("unknown id")
			return
		}
		if w.isTimedOut(first, timeoutChannel, application.Id) {
			w.acknowledgeMessage(application.Id)
			return
		}
		applicationResponse, ok := w.queryBank(application.Id)
		if !ok {
			continue //try again, later
		}
		if bankingsdk.ApplicationProcessed(applicationResponse.Status) {
			ok := w.serializeAndPublish(applicationResponse)
			if !ok {
				continue
			}
			w.acknowledgeMessage(application.Id)
			log.Infof("Processed application; id = %v", application.Id)
			break
		}
	}
}

func (w *Workload) isTimedOut(isFirst <-chan time.Time, timeout <-chan time.Time, id string) bool {
	select {
	case <-isFirst:
		log.Infof("Received application; id = %v", id)
	case <-time.After(w.RetryInterval):
		log.Debugf("Application not yet processed by bank; id = %v", id)
	case <-timeout:
		log.Infof("Workload timed out; id = %v", id)
		return true
	}
	return false
}

func (w *Workload) acknowledgeMessage(id string) {
	err := w.Message.Acknowledge()
	if err != nil {
		log.Errorf("acknowledgeMessage: Acknowledge failed for application; id = %v", id)
	}
}

func (w *Workload) queryBank(id string) (*bankingsdk.Application, bool) {
	applicationResponse, err := w.Bank.CheckStatus(id)
	if err != nil {
		log.Debugf("Execute: Cannot fetch application status; ApplicationId = %v", id)
		return &bankingsdk.Application{}, false
	}
	return applicationResponse, true
}

func (w *Workload) deserializeMessage() (*bankingsdk.Application, bool) {
	application, err := bankingsdk.DeserializeToApplication(w.Message.Body)
	if err != nil {
		log.Errorf("deserializeMessage: Cannot convert message, skipping it; error: %v", err)
		return &bankingsdk.Application{}, false
	} else {
		return application, true
	}
}

func (w *Workload) serializeAndPublish(application *bankingsdk.Application) bool {
	protoBytes, marshalError := bankingsdk.SerializeFromApplication(application)
	if marshalError != nil {
		log.Errorf("serializeFromMessage: Cannot convert message to bytes, skipping it; error: %v", marshalError)
		return false
	}

	publishError := w.Pub.Publish(protoBytes)
	if publishError != nil {
		log.Errorf("serializeAndPublish: Error publishing message; application_id = %v ;error = %v",
			application.Id, publishError)
		return false
	}
	return true
}
