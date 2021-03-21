package job

import (
	"bytes"
	"fmt"
	bankingsdk "github.com/aeolus3000/lendo-sdk/banking"
	"github.com/aeolus3000/lendo-sdk/messaging"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"strings"
	"time"
)

type Workload struct {
	Pub messaging.Publisher
	Bank bankingsdk.BankingApi
	Message *messaging.Message
	workloadTimeout time.Duration
	retryInterval time.Duration
}

func (w *Workload) Execute() {
	timeoutChannel := time.After(w.workloadTimeout)
	for {
		application, ok := deserializeMessage(w.Message)
		if !ok {
			return
		}
		applicationResponse, ok := queryBank(w.Bank, application.Id)
		if !ok {
			continue
		}
		if applicationProcessed(applicationResponse.Status) {
			publishMessage(applicationResponse, w.Message.Acknowledge, w.Pub)
			break
		}
		select {
		case <-time.After(w.retryInterval):
		case <-timeoutChannel:
			break
		}
	}
}

func convertToApplication(protoBytesBuffer bytes.Buffer, response *bankingsdk.Application) error {
	err := proto.Unmarshal(protoBytesBuffer.Bytes(), response)
	response.Status = strings.ToLower(response.Status)
	if err != nil {
		return fmt.Errorf("wrong response body format: %v; body: %s", err, string(protoBytesBuffer.Bytes()))
	}
	return nil
}

func applicationProcessed(status string) bool {
	if strings.Contains(status, bankingsdk.STATUS_COMPLETED) ||
		strings.Contains(status, bankingsdk.STATUS_REJECTED) {
		return true
	}
	return false
}

func queryBank(bank bankingsdk.BankingApi, id string) (*bankingsdk.Application, bool) {
	applicationResponse, err := bank.CheckStatus(id)
	if err != nil {
		log.Debugf("Execute: Cannot fetch application status; ApplicationId = ", id)
		return &bankingsdk.Application{}, false
	}
	return &applicationResponse, true
}

func deserializeMessage(message *messaging.Message) (*bankingsdk.Application, bool) {
	application := bankingsdk.Application{}
	err := convertToApplication(message.Body, &application)
	if err != nil {
		log.Error("deserializeMessage: Cannot convert message, skipping it; error: %v", err)
		ackErr := message.Acknowledge()
		log.Error("deserializeMessage: Can not ack message; error: %v", ackErr)
		return &bankingsdk.Application{}, false
	} else {
		return &application, true
	}
}

func serializeMessage(application *bankingsdk.Application, ack messaging.AcknowledgeFunc) (*bytes.Buffer, bool) {
	protoBytes, err := proto.Marshal(application)
	if err != nil {
		log.Error("serializeMessage: Cannot convert message to bytes, skipping it; error: %v", err)
		ackErr := ack()
		log.Error("serializeMessage: Can not ack message; error: %v", ackErr)
		return bytes.NewBuffer([]byte{}), false
	} else {
		return bytes.NewBuffer(protoBytes), true
	}
}

func publishMessage(application *bankingsdk.Application, ack messaging.AcknowledgeFunc, publisher messaging.Publisher) bool {
	buffer, ok := serializeMessage(application, ack)
	if !ok {
		return false
	}
	err := publisher.Publish(*buffer)
	if err != nil {
		log.Error("publishMessage: Error publishing message; application_id = %v ;error = %v", application.Id, err)
		return false
	}
	ackErr := ack()
	if ackErr != nil {
		log.Error("publishMessage: Can not ack message; error: %v", ackErr)
	}
	return true
}