package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/bnc-projects/ecs-instance-drainer/drainer"
	"github.com/bnc-projects/ecs-instance-drainer/message"
	"github.com/bnc-projects/ecs-instance-drainer/worker"
)

var AwsConfig aws.Config

func init() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatalf("Could not retrieve AWS configuration, %v", err)
	}
	AwsConfig = cfg
}

func main() {
	gracefulStop := make(chan os.Signal, 1)
	go stop(gracefulStop)

	messageChannel := make(chan message.LifecycleMessage, 10)
	metadataSvc := ec2metadata.New(AwsConfig)
	identity, err := metadataSvc.GetInstanceIdentityDocument()
	if err != nil {
		log.Fatal("Unable to get instance metadata")
	}
	AwsConfig.Region = identity.Region
	d := drainer.NewDrainer()
	sqs := sqs.New(AwsConfig)
	var wg sync.WaitGroup
	wg.Add(1)

	mp := message.NewMessageProcessor(messageChannel)
	mp.Start(sqs)
	w := worker.NewWorker(messageChannel)
	w.Start(d, ecs.New(AwsConfig), autoscaling.New(AwsConfig), sqs)

	wg.Wait()
	log.Print("Done")
}

func stop(gracefulStop chan os.Signal) {
	signal.Notify(gracefulStop, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Stopping due to %s", <-gracefulStop)
	os.Exit(0)
}
