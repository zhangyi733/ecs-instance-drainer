package drainer

import (
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/bnc-projects/ecs-instance-drainer/helper"
)

const (
	clusterEnv = "CLUSTER"
)

type Drainer struct {
	cluster             string
	InstanceId          string
	containerInstanceId string
}

type Drain interface {
	SetInstanceToDrain(svc ecsiface.ECSAPI) (bool, error)
	HasNoRunningTasks(svc ecsiface.ECSAPI) bool
	SetInstanceId(instanceId string)
}

func NewDrainer() *Drainer {
	return &Drainer{
		cluster: helper.EnvMustHave(clusterEnv),
	}
}

func (d *Drainer) HasNoRunningTasks(svc ecsiface.ECSAPI) bool {
	req := svc.ListTasksRequest(&ecs.ListTasksInput{
		Cluster:           &d.cluster,
		ContainerInstance: &d.containerInstanceId,
	})
	resp, err := req.Send()
	if err != nil {
		return false
	}
	if len(resp.TaskArns) > 0 {
		return false
	}
	return true
}

func (d *Drainer) SetInstanceToDrain(svc ecsiface.ECSAPI) (bool, error) {
	log.Printf("Attempting to drain instance %s", d.InstanceId)
	containerInstanceId, err := d.findInstanceToDrain(svc)
	if err != nil {
		log.Printf("Could not find instance id %s to drain.", d.InstanceId)
		return false, err
	}
	d.containerInstanceId = containerInstanceId
	req := svc.UpdateContainerInstancesStateRequest(&ecs.UpdateContainerInstancesStateInput{
		Cluster:            &d.cluster,
		ContainerInstances: []string{containerInstanceId},
		Status:             ecs.ContainerInstanceStatusDraining,
	})
	_, err = req.Send()
	if err != nil {
		log.Printf("Error %v setting message timeout", err)
		return false, err
	}
	log.Printf("Successfully draining instance %s", d.InstanceId)
	return true, nil
}

func (d *Drainer) findInstanceToDrain(svc ecsiface.ECSAPI) (string, error) {
	req := svc.ListContainerInstancesRequest(&ecs.ListContainerInstancesInput{Cluster: &d.cluster})
	resp, err := req.Send()
	if err != nil {
		log.Printf("Error retrieving container instances %v", err)
		return "", err
	}
	for _, containerInstanceArn := range resp.ContainerInstanceArns {
		describeReq := svc.DescribeContainerInstancesRequest(&ecs.DescribeContainerInstancesInput{
			Cluster:            &d.cluster,
			ContainerInstances: []string{containerInstanceArn},
		})
		resp, err := describeReq.Send()
		if err != nil {
			log.Printf("Could not get information about container instance %v", err)
		} else {
			for _, containerInstance := range resp.ContainerInstances {
				if *containerInstance.Ec2InstanceId == d.InstanceId {
					log.Printf("Found container instance %s for instance %s", containerInstanceArn, d.InstanceId)
					return *containerInstance.ContainerInstanceArn, nil
				}
			}
		}
	}
	return "", errors.New("Could not find container instance")
}

func (d *Drainer) SetInstanceId(instanceId string) {
	d.InstanceId = instanceId
}
