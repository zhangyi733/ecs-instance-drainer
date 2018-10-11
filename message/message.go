package message

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/bnc-projects/ecs-instance-drainer/helper"
)

const (
	queueEnvVar       = "LIFECYCLE_QUEUE"
	sqsWaitSeconds    = int64(20)
	visibilityTimeout = int64(480)
)

type LifecycleMessage struct {
	AccountId            string `json:"AccountId"`
	RequestId            string `json:"RequestId"`
	AutoScalingGroupName string `json:"AutoScalingGroupName"`
	LifecycleTransition  string `json:"LifecycleTransition"`
	LifecycleHookName    string `json:"LifecycleHookName"`
	EC2InstanceId        string `json:"EC2InstanceId"`
	LifecycleActionToken string `json:"LifecycleActionToken"`
	ReceiptHandle        string `json:"ReceiptHandle,omitempty"`
}

type MessageProccesor struct {
	Message  chan LifecycleMessage
	queueUrl string
	quit     chan bool
}

type Message interface {
	Start(svc sqsiface.SQSAPI)
	RetrieveMessages()
}

func NewMessageProcessor(messageChannel chan LifecycleMessage) *MessageProccesor {
	return &MessageProccesor{
		Message:  messageChannel,
		queueUrl: helper.EnvMustHave(queueEnvVar),
		quit:     make(chan bool),
	}
}

func (mp *MessageProccesor) Start(svc sqsiface.SQSAPI) {
	go func() {
		for {
			select {
			case <-mp.quit:
				return
			default:
				mp.RetrieveMessage(svc)
			}
		}
	}()
}

func (mp *MessageProccesor) RetrieveMessage(svc sqsiface.SQSAPI) {
	w := sqsWaitSeconds
	vs := visibilityTimeout
	req := svc.ReceiveMessageRequest(&sqs.ReceiveMessageInput{
		QueueUrl:          &mp.queueUrl,
		WaitTimeSeconds:   &w,
		VisibilityTimeout: &vs,
	})
	resp, err := req.Send()
	if err != nil {
		log.Printf("%v sending get messages request", err)
		return
	}
	for _, message := range resp.Messages {
		lc := new(LifecycleMessage)
		err := json.Unmarshal([]byte(*message.Body), &lc)
		if err != nil {
			log.Printf("Error parsing message %v", message)
		}
		lc.ReceiptHandle = *message.ReceiptHandle
		mp.Message <- *lc
	}
}

func (mp *MessageProccesor) stop() {
	go func() {
		mp.quit <- true
	}()
}
