package worker

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/bnc-projects/ecs-instance-drainer/drainer"
	"github.com/bnc-projects/ecs-instance-drainer/helper"
	"github.com/bnc-projects/ecs-instance-drainer/message"
	"github.com/facebookgo/clock"
)

const (
	timeout     = "DRAINER_TIMEOUT"
	queueEnvVar = "LIFECYCLE_QUEUE"
)

type Worker struct {
	Message chan message.LifecycleMessage
	drainer *drainer.Drainer
	quit    chan bool
}

func NewWorker(messageChannel chan message.LifecycleMessage) *Worker {
	return &Worker{
		Message: messageChannel,
		quit:    make(chan bool),
	}
}

func (w *Worker) Start(d drainer.Drain, ecsService ecsiface.ECSAPI, asgService autoscalingiface.AutoScalingAPI, sqsService sqsiface.SQSAPI) {
	go func() {
		for {
			select {
			case msg := <-w.Message:
				go func() {
					d.SetInstanceId(msg.EC2InstanceId)
					log.Printf("Got Message: '%v'", msg)
					_, err := d.SetInstanceToDrain(ecsService)
					if err != nil {
						log.Print("Error setting instance to draining")
						// Mostly likely caused by the instance being terminated so we will delete the message.
						DeleteMessage(msg.ReceiptHandle, sqsService)
					} else {
						terminated := w.terminateNode(clock.New().Ticker(30*time.Second), d, ecsService, asgService, msg)
						if terminated {
							log.Printf("Deleting message %s for instance %s", msg.RequestId, msg.EC2InstanceId)
							DeleteMessage(msg.ReceiptHandle, sqsService)
							w.quit <- true
						}
					}
				}()
			case <-w.quit:
				return
			}
		}
	}()
}

func (w *Worker) terminateNode(ticker *clock.Ticker, d drainer.Drain, ecsService ecsiface.ECSAPI, asgService autoscalingiface.AutoScalingAPI, msg message.LifecycleMessage) bool {
	now := <-ticker.C
	v := helper.EnvMustHave(timeout)
	timeout, err := time.ParseDuration(v)
	if err != nil {
		log.Print("Could not parse timeout, defaulting to 1 hour")
		timeout = time.Duration(1 * time.Hour)
	}
	end := now.Add(timeout)
	done := make(chan bool, 1)
	for {
		select {
		case d := <-done:
			ticker.Stop()
			return d
		case t := <-ticker.C:
			if t.After(end) {
				log.Printf("Could not shutdown instance within timeout period")
				done <- false
			}
			if d.HasNoRunningTasks(ecsService) {
				log.Printf("No tasks running, shutting down %s", msg.EC2InstanceId)
				req := asgService.CompleteLifecycleActionRequest(&autoscaling.CompleteLifecycleActionInput{
					AutoScalingGroupName:  &msg.AutoScalingGroupName,
					InstanceId:            &msg.EC2InstanceId,
					LifecycleActionResult: aws.String("CONTINUE"),
					LifecycleActionToken:  &msg.LifecycleActionToken,
					LifecycleHookName:     &msg.LifecycleHookName,
				})
				_, err := req.Send()
				if err != nil {
					log.Printf("Failed to complete termination %v of instance %s, exiting", err, msg.EC2InstanceId)
					done <- false
				} else {
					log.Printf("Completed termination lifecycle of instance %s", msg.EC2InstanceId)
					done <- true
				}
			}
		}
	}
}

func DeleteMessage(receiptHandle string, svc sqsiface.SQSAPI) {
	queueUrl := helper.EnvMustHave(queueEnvVar)
	req := svc.DeleteMessageRequest(&sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: &receiptHandle,
	})
	_, err := req.Send()
	if err != nil {
		log.Printf("Could not delete message %s from SQS", receiptHandle)
	}
}

func (w *Worker) stop() {
	go func() {
		w.quit <- true
	}()
}
