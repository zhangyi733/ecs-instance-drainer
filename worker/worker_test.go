package worker

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/TechemyLtd/ecs-instance-drainer/drainer"
	"github.com/TechemyLtd/ecs-instance-drainer/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/facebookgo/clock"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("DRAINER_TIMEOUT", "")
}

func TestCreateWorker(t *testing.T) {
	messageChan := make(chan message.LifecycleMessage, 10)
	w := NewWorker(messageChan)

	assert.Equal(t, messageChan, w.Message)
	assert.NotNil(t, w.quit)
}

func TestWorkerGotMessage(t *testing.T) {
	messageChan := make(chan message.LifecycleMessage, 10)
	w := NewWorker(messageChan)

	mockECSSvc := mockECSClient{}
	mockASGSvc := &mockASGClient{}
	mockDrainer := &mockDrainerClient{}
	go func() {
		w.Start(mockDrainer, mockECSSvc, mockASGSvc)
	}()
	w.Message <- message.LifecycleMessage{LifecycleTransition: "Test Message"}
	w.stop()
}

func TestStop(t *testing.T) {
	messageChan := make(chan message.LifecycleMessage, 10)
	w := NewWorker(messageChan)

	mockECSSvc := mockECSClient{}
	mockASGSvc := &mockASGClient{}
	mockDrainer := &mockDrainerClient{}
	w.Start(mockDrainer, mockECSSvc, mockASGSvc)
	w.stop()

	assert.True(t, <-w.quit)
}

func TestTerminateNode(t *testing.T) {
	os.Setenv("DRAINER_TIMEOUT", "1h")
	defer os.Setenv("DRAINER_TIMEOUT", "")
	messageChan := make(chan message.LifecycleMessage, 10)
	w := NewWorker(messageChan)

	msg := message.LifecycleMessage{EC2InstanceId: "i-testinstanceid"}
	mockECSSvc := mockECSClient{}
	mockASGSvc := &mockASGClient{}
	mockDrainer := &mockDrainerClient{resp: true}
	mc := clock.NewMock()
	go func() {
		ticker := mc.Ticker(30 * time.Second)
		terminated := w.terminateNode(ticker, mockDrainer, mockECSSvc, mockASGSvc, msg)
		assert.True(t, terminated)
	}()
	runtime.Gosched()
	mc.Add(60 * time.Second)
}

func TestTerminateNodeTimeout(t *testing.T) {
	os.Setenv("DRAINER_TIMEOUT", "1h")
	defer os.Setenv("DRAINER_TIMEOUT", "")
	messageChan := make(chan message.LifecycleMessage, 10)
	w := NewWorker(messageChan)

	msg := message.LifecycleMessage{EC2InstanceId: "i-testinstanceid"}
	mockECSSvc := mockECSClient{}
	mockASGSvc := &mockASGClient{}
	mockDrainer := &mockDrainerClient{resp: false}
	mc := clock.NewMock()
	go func() {
		ticker := mc.Ticker(30 * time.Second)
		terminated := w.terminateNode(ticker, mockDrainer, mockECSSvc, mockASGSvc, msg)
		assert.False(t, terminated)
	}()
	runtime.Gosched()
	mc.Add(3660 * time.Second)
}

type mockECSClient struct {
	ecsiface.ECSAPI
}

type mockASGClient struct {
	autoscalingiface.AutoScalingAPI
}

type mockDrainerClient struct {
	resp bool
	drainer.Drain
}

func (m *mockDrainerClient) HasNoRunningTasks(svc ecsiface.ECSAPI) bool {
	return m.resp
}

func (m *mockASGClient) CompleteLifecycleActionRequest(input *autoscaling.CompleteLifecycleActionInput) autoscaling.CompleteLifecycleActionRequest {
	return autoscaling.CompleteLifecycleActionRequest{
		Request: &aws.Request{
			Data: &autoscaling.CompleteLifecycleActionOutput{},
		},
	}
}
