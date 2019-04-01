package message

import (
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("LIFECYCLE_QUEUE", "")
}

func TestCreateMessageProcessor(t *testing.T) {
	queueUrl := "message-queue"
	os.Setenv("LIFECYCLE_QUEUE", queueUrl)
	defer os.Setenv("LIFECYCLE_QUEUE", "")

	messageChan := make(chan LifecycleMessage, 10)
	mp := NewMessageProcessor(messageChan)

	assert.Equal(t, messageChan, mp.Message)
	assert.Equal(t, queueUrl, mp.queueUrl)
	assert.NotNil(t, mp.quit)
}

func TestNoMessages(t *testing.T) {
	os.Setenv("LIFECYCLE_QUEUE", "test-queue")
	defer os.Setenv("LIFECYCLE_QUEUE", "")
	mockSvc := &mockSQSClient{}
	messageChan := make(chan LifecycleMessage, 10)
	mp := NewMessageProcessor(messageChan)

	mp.RetrieveMessage(mockSvc)

	assert.Empty(t, mp.Message)
}

func TestRetrieveMessages(t *testing.T) {
	os.Setenv("LIFECYCLE_QUEUE", "test-queue")
	defer os.Setenv("LIFECYCLE_QUEUE", "")
	data := sqs.ReceiveMessageOutput{
		Messages: []sqs.Message{
			{
				MessageId:     aws.String(`7228930f-001b-4790-9fa5-3a04b2b860ec`),
				MD5OfBody:     aws.String(`eb0bf8297a9754cd4d7fd3b2126b7aca`),
				Body:          aws.String(`{"AccountId":"609519224176","RequestId":"b36d6757-57c7-11e8-a9ca-b3409db76d91","AutoScalingGroupARN":"arn:aws:autoscaling:us-west-2:609519224176:autoScalingGroup:d3ab46f5-9274-4f34-8141-87b0384f69ec:autoScalingGroupName/MarketDataECS-AutoScalingGroup-TZI0DMYKJBJG","AutoScalingGroupName":"MarketDataECS-AutoScalingGroup-TZI0DMYKJBJG","Service":"AWS Auto Scaling","Event":"autoscaling:TEST_NOTIFICATION","Time":"2018-05-14T22:39:45.555Z"}`),
				ReceiptHandle: aws.String(uuid.Must(uuid.NewV4()).String()),
			},
			{
				MessageId:     aws.String(`3091551c-cd47-442b-94a5-4d1ffecb667f `),
				MD5OfBody:     aws.String(`0cbc6611f5540bd0809a388dc95a615b`),
				Body:          aws.String(`{"LifecycleHookName":"MarketDataECS-AutoScalingGroupTerminatingLifecycleHook-65KCAZVO19BF","AccountId":"609519224176","RequestId":"5d858daa-5692-135e-98ca-ef04118c2ab5","LifecycleTransition":"autoscaling:EC2_INSTANCE_TERMINATING","AutoScalingGroupName":"MarketDataECS-AutoScalingGroup-TZI0DMYKJBJG","Service":"AWS Auto Scaling","Time":"2018-05-16T20:13:40.414Z","EC2InstanceId":"i-0194708ef7e24dcc7","LifecycleActionToken":"602063c7-4a06-4a67-b8cf-87ccc33bdc86"}`),
				ReceiptHandle: aws.String(uuid.Must(uuid.NewV4()).String()),
			},
			{
				MessageId:     aws.String(`c2851353-c7bc-469c-ad19-1e74bb4e11de`),
				MD5OfBody:     aws.String(`a508e4ce8f5b14771ffb4b2ffb853662`),
				Body:          aws.String(`{"LifecycleHookName":"MarketDataECS-AutoScalingGroupTerminatingLifecycleHook-65KCAZVO19BF","AccountId":"609519224176","RequestId":"5d858daa-5692-135e-98ca-ef04118c2ab5","LifecycleTransition":"autoscaling:EC2_INSTANCE_TERMINATING","AutoScalingGroupName":"MarketDataECS-AutoScalingGroup-TZI0DMYKJBJG","Service":"AWS Auto Scaling","Time":"2018-05-16T20:13:40.414Z","EC2InstanceId":"i-0194708ef7e24dcc7","LifecycleActionToken":"602063c7-4a06-4a67-b8cf-87ccc33bdc86"}`),
				ReceiptHandle: aws.String(uuid.Must(uuid.NewV4()).String()),
			},
		},
	}
	mockSvc := &mockSQSClient{
		Resp: data,
	}
	messageChan := make(chan LifecycleMessage, 10)
	mp := NewMessageProcessor(messageChan)

	mp.RetrieveMessage(mockSvc)

	assert.NotEmpty(t, mp.Message)
	assert.Equal(t, 3, len(mp.Message))
}

func TestRetrieveInvalidMessage(t *testing.T) {
	os.Setenv("LIFECYCLE_QUEUE", "test-queue")
	defer os.Setenv("LIFECYCLE_QUEUE", "")
	data := sqs.ReceiveMessageOutput{
		Messages: []sqs.Message{
			{
				MessageId:     aws.String(`7228930f-001b-4790-9fa5-3a04b2b860ec`),
				MD5OfBody:     aws.String(`eb0bf8297a9754cd4d7fd3b2126b7aca`),
				Body:          aws.String(`Test Invalid Message`),
				ReceiptHandle: aws.String(uuid.Must(uuid.NewV4()).String()),
			},
		},
	}
	mockSvc := &mockSQSClient{
		Resp: data,
	}
	messageChan := make(chan LifecycleMessage, 10)
	mp := NewMessageProcessor(messageChan)

	mp.RetrieveMessage(mockSvc)

	m := <-mp.Message
	assert.Empty(t, m.LifecycleTransition)
}

func TestErrorOnRetrieve(t *testing.T) {
	os.Setenv("LIFECYCLE_QUEUE", "fail-send-queue")
	defer os.Setenv("LIFECYCLE_QUEUE", "")
	mockSvc := &mockSQSClient{}
	messageChan := make(chan LifecycleMessage, 10)
	mp := NewMessageProcessor(messageChan)

	mp.RetrieveMessage(mockSvc)

	assert.Empty(t, mp.Message)
}

func TestStop(t *testing.T) {
	os.Setenv("LIFECYCLE_QUEUE", "test-queue")
	defer os.Setenv("LIFECYCLE_QUEUE", "")
	mockSvc := &mockSQSClient{}
	messageChan := make(chan LifecycleMessage, 10)
	mp := NewMessageProcessor(messageChan)

	mp.Start(mockSvc)
	mp.stop()

	assert.True(t, <-mp.quit)
}

type mockSQSClient struct {
	sqsiface.SQSAPI
	Resp sqs.ReceiveMessageOutput
}

func (m *mockSQSClient) AddPermission(input *sqs.AddPermissionInput) (*sqs.AddPermissionOutput, error) {
	return nil, nil
}

func (m *mockSQSClient) ReceiveMessageRequest(input *sqs.ReceiveMessageInput) sqs.ReceiveMessageRequest {
	if *input.QueueUrl == "fail-send-queue" {
		return sqs.ReceiveMessageRequest{
			Request: &aws.Request{
				Data:  &m.Resp,
				Error: errors.New("Error: Receive Message Request"),
			},
		}
	}
	return sqs.ReceiveMessageRequest{
		Request: &aws.Request{
			Data: &m.Resp,
		},
	}
}
