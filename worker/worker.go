package worker

import (
	"log"
	"time"

	"github.com/TechemyLtd/ecs-instance-drainer/drainer"
	"github.com/TechemyLtd/ecs-instance-drainer/helper"
	"github.com/TechemyLtd/ecs-instance-drainer/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/facebookgo/clock"
)

const (
	timeout = "DRAINER_TIMEOUT"
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

func (w *Worker) Start(d drainer.Drain, ecsService ecsiface.ECSAPI, asgService autoscalingiface.AutoScalingAPI) {
	go func() {
		for {
			select {
			case msg := <-w.Message:
				log.Printf("Got Message: '%v'", msg)
				if msg.EC2InstanceId != "" {
					_, err := d.SetInstanceToDrain(ecsService)
					if err != nil {
						log.Print("Error setting instance to draining")
						return
					}
					terminated := w.terminateNode(clock.New().Ticker(30*time.Second), d, ecsService, asgService, msg)
					if terminated {
						// TODO Delete the SQS Message
						w.quit <- true
					}
				}
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
				req := asgService.CompleteLifecycleActionRequest(&autoscaling.CompleteLifecycleActionInput{
					AutoScalingGroupName:  &msg.AutoScalingGroupName,
					InstanceId:            &msg.EC2InstanceId,
					LifecycleActionResult: aws.String("CONTINUE"),
					LifecycleActionToken:  &msg.LifecycleActionToken,
					LifecycleHookName:     &msg.LifecycleHookName,
				})
				_, err := req.Send()
				if err == nil {
					log.Printf("Completed termination lifecycle of %s instance", msg.EC2InstanceId)
					done <- true
				}
			}
		}
	}
}

func (w *Worker) stop() {
	go func() {
		w.quit <- true
	}()
}
