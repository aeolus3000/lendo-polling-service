package job

import (
	"bytes"
	"fmt"
	bankingsdk "github.com/aeolus3000/lendo-sdk/banking"
	"github.com/aeolus3000/lendo-sdk/messaging"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"reflect"
	"testing"
	"time"
)

func TestWorkload_isTimedOut(t *testing.T) {
	tests := []struct {
		name                 string
		timeOutSeconds       time.Duration
		firstTime            time.Duration
		retryTime            time.Duration
		expectedMilliseconds int64
		expectedTimedOut     bool
	}{
		{
			"timeout",
			1 * time.Second,
			5 * time.Second,
			5 * time.Second,
			1030,
			true,
		}, {
			"first",
			5 * time.Second,
			1 * time.Second,
			5 * time.Second,
			1030,
			false,
		}, {
			"retry",
			5 * time.Second,
			5 * time.Second,
			2 * time.Second,
			2030,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeOutChannel := time.After(tt.timeOutSeconds)
			first := time.After(tt.firstTime)
			w := &Workload{
				Pub:             nil,
				Bank:            nil,
				Message:         nil,
				WorkloadTimeout: tt.timeOutSeconds,
				RetryInterval:   tt.retryTime,
			}
			start := time.Now()
			timedOut := w.isTimedOut(first, timeOutChannel, "any id")
			elapsed := time.Since(start)
			if elapsed.Milliseconds() > tt.expectedMilliseconds {
				t.Error("Method returned too early")
			}
			if timedOut != tt.expectedTimedOut {
				t.Errorf("Expected timed out = %v; Got timed out = %v", timedOut, tt.expectedTimedOut)
			}
		})
	}

}

type TestAcknowledge struct {
	acknowledged bool
}

func (ta *TestAcknowledge) acknowledge() error {
	ta.acknowledged = true
	return nil
}

func TestWorkload_acknowledgeMessage(t *testing.T) {

	testAcknowledge := TestAcknowledge{false}
	w := &Workload{
		Message: &messaging.Message{Acknowledge: testAcknowledge.acknowledge},
	}
	w.acknowledgeMessage("some id")
	if testAcknowledge.acknowledged != true {
		t.Errorf("Acknowledge Method was not called")
	}
}

type TestBank struct {
	application *bankingsdk.Application
	err         error
}

func (tb *TestBank) CheckStatus(id string) (*bankingsdk.Application, error) {
	if tb.err != nil {
		return nil, tb.err
	}
	return tb.application, nil
}

func (tb *TestBank) Create(application *bankingsdk.Application) (*bankingsdk.Application, error) {
	panic("Create() should not have been called")
}

func TestWorkload_queryBank(t *testing.T) {
	expectedApplication := bankingsdk.Application{
		Id:        uuid.NewString(),
		FirstName: "Employer",
		LastName:  "Of the month",
		Status:    "pending",
		JobId:     uuid.NewString(),
	}
	tests := []struct {
		name                string
		actualError         error
		expectedApplication *bankingsdk.Application
	}{
		{
			"no error",
			nil,
			&expectedApplication,
		}, {
			"some error",
			fmt.Errorf("having trouble checking the application"),
			&expectedApplication,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testBank := &TestBank{
				application: tt.expectedApplication,
				err:         tt.actualError,
			}
			w := &Workload{
				Bank: testBank,
			}
			gotApplication, gotOk := w.queryBank("some id")
			if tt.actualError != nil && gotOk == true {
				t.Error("queryBank() was successful but was expected to be not successful")
			}
			if tt.actualError == nil && gotOk == false {
				t.Error("queryBank() was not successful but was expected to be successful")
			}
			if tt.actualError == nil && !proto.Equal(gotApplication, tt.expectedApplication) {
				t.Error("queryBank() got and want Application are different")
			}
		})
	}
}

type TestPub struct {
	bytesBuffer *bytes.Buffer
	err         error
}

func (tp *TestPub) Publish(buffer *bytes.Buffer) error {
	if tp.err != nil {
		return tp.err
	}
	tp.bytesBuffer = buffer
	return nil
}

func (tp *TestPub) Close() error {
	panic("Close() should not have been called")
}

func TestWorkload_serializeAndPublish(t *testing.T) {
	application := bankingsdk.Application{
		Id:        "8cb9fd11-fc05-45d2-93fa-23f62dee24b2",
		FirstName: "Employer",
		LastName:  "Of the month",
		Status:    "pending",
		JobId:     "8cb9fd11-fc05-45d2-93fa-23f62dee24b2",
	}
	applicationBytes := bytes.NewBuffer([]byte{10, 36, 56, 99, 98, 57, 102, 100, 49, 49, 45, 102, 99, 48, 53, 45, 52, 53, 100, 50, 45, 57,
		51, 102, 97, 45, 50, 51, 102, 54, 50, 100, 101, 101, 50, 52, 98, 50, 18, 8, 69, 109, 112, 108, 111, 121, 101, 114, 26, 12, 79, 102, 32, 116,
		104, 101, 32, 109, 111, 110, 116, 104, 34, 7, 112, 101, 110, 100, 105, 110, 103, 42, 36, 56, 99, 98, 57, 102, 100, 49, 49, 45, 102, 99, 48,
		53, 45, 52, 53, 100, 50, 45, 57, 51, 102, 97, 45, 50, 51, 102, 54, 50, 100, 101, 101, 50, 52, 98, 50})
	type args struct {
		application *bankingsdk.Application
	}
	tests := []struct {
		name      string
		err       error
		wantBytes *bytes.Buffer
	}{
		{
			"no error",
			nil,
			applicationBytes,
		},
		{
			"error",
			fmt.Errorf("having trouble serializing the application"),
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPub := TestPub{
				err: tt.err,
			}
			w := &Workload{
				Pub: &testPub,
			}
			ok := w.serializeAndPublish(&application)
			if tt.err != nil && ok == true {
				t.Error("serializeAndPublish() was successful but was expected to be not successful")
			}
			if tt.err == nil && ok == false {
				t.Errorf("serializeAndPublish() was not successful but was expected to be successful")
			}
			if tt.err == nil && !reflect.DeepEqual(testPub.bytesBuffer.Bytes(), tt.wantBytes.Bytes()) {
				fmt.Println(testPub.bytesBuffer.Bytes())
				fmt.Println(tt.wantBytes.Bytes())
				t.Errorf("serializeAndPublish() got and want Application bytes are different")
			}
		})
	}
}

func TestWorkload_deserializeMessage(t *testing.T) {
	expectedApplication := bankingsdk.Application{
		Id:        "8cb9fd11-fc05-45d2-93fa-23f62dee24b2",
		FirstName: "Employer",
		LastName:  "Of the month",
		Status:    "pending",
		JobId:     "8cb9fd11-fc05-45d2-93fa-23f62dee24b2",
	}
	applicationBytes := bytes.NewBuffer([]byte{10, 36, 56, 99, 98, 57, 102, 100, 49, 49, 45, 102, 99, 48, 53, 45, 52, 53, 100, 50, 45, 57,
		51, 102, 97, 45, 50, 51, 102, 54, 50, 100, 101, 101, 50, 52, 98, 50, 18, 8, 69, 109, 112, 108, 111, 121, 101, 114, 26, 12, 79, 102, 32, 116,
		104, 101, 32, 109, 111, 110, 116, 104, 34, 7, 112, 101, 110, 100, 105, 110, 103, 42, 36, 56, 99, 98, 57, 102, 100, 49, 49, 45, 102, 99, 48,
		53, 45, 52, 53, 100, 50, 45, 57, 51, 102, 97, 45, 50, 51, 102, 54, 50, 100, 101, 101, 50, 52, 98, 50})
	applicationBytes2 := bytes.NewBuffer([]byte{10, 36, 56, 99, 98, 57, 102, 100, 49, 49, 45, 102, 99, 48, 53, 45, 52, 53, 100, 50, 45, 57, 51, 102, 97, 45, 50, 51, 102,
		54, 50, 100, 101, 101, 50, 52, 98, 50, 18, 8, 69, 109, 112, 108, 111, 121, 101, 114, 26, 12, 79, 102, 32, 116, 104, 101, 32, 109, 111, 110,
		116, 104, 34, 7, 80, 69, 78, 68, 73, 78, 71, 42, 36, 56, 99, 98, 57, 102, 100, 49, 49, 45, 102, 99, 48, 53, 45, 52, 53, 100, 50, 45, 57, 51, 102,
		97, 45, 50, 51, 102, 54, 50, 100, 101, 101, 50, 52, 98, 50}) //status "PENDING" in capital letters -> deserialize should lower case
	tests := []struct {
		name             string
		applicationBytes *bytes.Buffer
		wantOk           bool
	}{
		{
			"ok",
			applicationBytes,
			true,
		}, {
			"ok, should make to lower on status",
			applicationBytes2,
			true,
		}, {
			"error",
			bytes.NewBuffer([]byte{10, 36, 56, 99, 98, 57, 102}),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := messaging.Message{
				Body: tt.applicationBytes,
			}
			w := &Workload{
				Message: &message,
			}
			application, ok := w.deserializeMessage()
			if ok != tt.wantOk {
				t.Errorf("deserializeMessage() gotOk = %v, wantOk %v", ok, tt.wantOk)
			}
			if !tt.wantOk && !proto.Equal(application, &bankingsdk.Application{}) {
				t.Error("deserializeMessage() application wasn't empty in case of unsuccesful deserialization")
			}
			if tt.wantOk && !proto.Equal(application, &expectedApplication) {
				t.Error("deserializeMessage() applications didn't match")
			}
		})
	}
}
