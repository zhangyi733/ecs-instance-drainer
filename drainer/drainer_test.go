package drainer

import (
	"errors"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/ecsiface"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("CLUSTER", "")
}

func TestSetInstanceId(t *testing.T) {
	os.Setenv("CLUSTER", "test-cluster")
	defer os.Setenv("CLUSTER", "")

	expected := "testInstanceId"
	d := NewDrainer()
	d.SetInstanceId(expected)
	assert.Equal(t, expected, d.InstanceId)
}

func TestHasNoRunningTasks(t *testing.T) {
	os.Setenv("CLUSTER", "test-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	d.containerInstanceId = "testContainerInstanceArn"

	data := ecs.ListTasksOutput{
		TaskArns: []string{},
	}
	mockSvc := &mockECSClient{ListTaskResp: data}
	actual := d.HasNoRunningTasks(mockSvc)

	assert.True(t, actual)
}

func TestHasRunningTasks(t *testing.T) {
	os.Setenv("CLUSTER", "test-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	d.containerInstanceId = "testContainerInstanceArn"

	data := ecs.ListTasksOutput{
		TaskArns: []string{
			"taskArn",
		},
	}
	mockSvc := &mockECSClient{ListTaskResp: data}
	actual := d.HasNoRunningTasks(mockSvc)

	assert.False(t, actual)
}

func TestHasRunningTasksFailedRequest(t *testing.T) {
	os.Setenv("CLUSTER", "invalid-request-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	d.containerInstanceId = "testContainerInstanceArn"

	mockSvc := &mockECSClient{}
	actual := d.HasNoRunningTasks(mockSvc)

	assert.False(t, actual)
}

func TestFindInstanceToDrain(t *testing.T) {
	os.Setenv("CLUSTER", "test-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	containerArn := "testContainerInstanceArn"
	listData := ecs.ListContainerInstancesOutput{
		ContainerInstanceArns: []string{
			containerArn,
			"testArn2",
		},
	}
	containerInstance := ecs.ContainerInstance{
		ContainerInstanceArn: &containerArn,
		Ec2InstanceId:        &d.InstanceId,
	}
	data := ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []ecs.ContainerInstance{
			containerInstance,
		},
	}
	mockSvc := &mockECSClient{ListContainerResp: listData, ContainerInstanceResp: data}
	containerId, _ := d.findInstanceToDrain(mockSvc)
	assert.Equal(t, containerArn, containerId)
}

func TestFindInstanceToDrainUnknownCluster(t *testing.T) {
	os.Setenv("CLUSTER", "invalid-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	mockSvc := &mockECSClient{}
	_, err := d.findInstanceToDrain(mockSvc)
	assert.Error(t, err)
}

func TestFindInstanceToDrainNoContainers(t *testing.T) {
	os.Setenv("CLUSTER", "invalid-cluster-describe-instances")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	containerArn := "testContainerInstanceArn"
	listData := ecs.ListContainerInstancesOutput{
		ContainerInstanceArns: []string{
			containerArn,
			"testArn2",
		},
	}
	containerInstance := ecs.ContainerInstance{
		ContainerInstanceArn: &containerArn,
		Ec2InstanceId:        &d.InstanceId,
	}
	data := ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []ecs.ContainerInstance{
			containerInstance,
		},
	}
	mockSvc := &mockECSClient{ListContainerResp: listData, ContainerInstanceResp: data}
	_, err := d.findInstanceToDrain(mockSvc)
	assert.Error(t, err)
}

func TestDrainECSInstanceContainerNotFound(t *testing.T) {
	os.Setenv("CLUSTER", "test-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	mockSvc := &mockECSClient{}
	_, err := d.SetInstanceToDrain(mockSvc)
	assert.Error(t, err)
}

func TestDrainECSInstanceFail(t *testing.T) {
	os.Setenv("CLUSTER", "update-state-fail-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	containerArn := "testContainerArn"
	listData := ecs.ListContainerInstancesOutput{
		ContainerInstanceArns: []string{
			containerArn,
		},
	}
	containerInstance := ecs.ContainerInstance{
		ContainerInstanceArn: &containerArn,
		Ec2InstanceId:        &d.InstanceId,
	}
	data := ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []ecs.ContainerInstance{
			containerInstance,
		},
	}
	mockSvc := &mockECSClient{ListContainerResp: listData, ContainerInstanceResp: data}
	updated, err := d.SetInstanceToDrain(mockSvc)
	assert.False(t, updated)
	assert.Error(t, err)
}

func TestDrainECSInstance(t *testing.T) {
	os.Setenv("CLUSTER", "test-cluster")
	defer os.Setenv("CLUSTER", "")

	d := NewDrainer()
	d.InstanceId = "testInstanceId"
	containerArn := "testContainerArn"
	listData := ecs.ListContainerInstancesOutput{
		ContainerInstanceArns: []string{
			containerArn,
		},
	}
	containerInstance := ecs.ContainerInstance{
		ContainerInstanceArn: &containerArn,
		Ec2InstanceId:        &d.InstanceId,
	}
	data := ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []ecs.ContainerInstance{
			containerInstance,
		},
	}
	mockSvc := &mockECSClient{ListContainerResp: listData, ContainerInstanceResp: data}
	updated, err := d.SetInstanceToDrain(mockSvc)
	assert.Equal(t, containerArn, d.containerInstanceId)
	assert.True(t, updated)
	assert.Nil(t, err)
}

type mockECSClient struct {
	ecsiface.ECSAPI
	ListContainerResp     ecs.ListContainerInstancesOutput
	ContainerInstanceResp ecs.DescribeContainerInstancesOutput
	UpdateStateResp       ecs.UpdateContainerInstancesStateOutput
	ListTaskResp          ecs.ListTasksOutput
}

func (m *mockECSClient) ListContainerInstancesRequest(input *ecs.ListContainerInstancesInput) ecs.ListContainerInstancesRequest {
	if *input.Cluster == "invalid-cluster" {
		return ecs.ListContainerInstancesRequest{
			Request: &aws.Request{
				Data:  &ecs.ListContainerInstancesOutput{},
				Error: errors.New("Error: List Contqainer Instances Request"),
			},
		}
	}
	return ecs.ListContainerInstancesRequest{
		Request: &aws.Request{
			Data: &m.ListContainerResp,
		},
	}
}

func (m *mockECSClient) DescribeContainerInstancesRequest(input *ecs.DescribeContainerInstancesInput) ecs.DescribeContainerInstancesRequest {
	if *input.Cluster == "invalid-cluster-describe-instances" {
		log.Printf("Cluster is %s", *input.Cluster)
		return ecs.DescribeContainerInstancesRequest{
			Request: &aws.Request{
				Data:  &ecs.DescribeContainerInstancesOutput{},
				Error: errors.New("Error: Describe Container Instances Request"),
			},
		}
	}
	return ecs.DescribeContainerInstancesRequest{
		Request: &aws.Request{
			Data: &m.ContainerInstanceResp,
		},
	}
}

func (m *mockECSClient) UpdateContainerInstancesStateRequest(input *ecs.UpdateContainerInstancesStateInput) ecs.UpdateContainerInstancesStateRequest {
	if *input.Cluster == "update-state-fail-cluster" {
		return ecs.UpdateContainerInstancesStateRequest{
			Request: &aws.Request{
				Data:  &ecs.UpdateContainerInstancesStateOutput{},
				Error: errors.New("Error: Update Container Instance State"),
			},
		}
	}
	return ecs.UpdateContainerInstancesStateRequest{
		Request: &aws.Request{
			Data: &m.UpdateStateResp,
		},
	}
}

func (m *mockECSClient) ListTasksRequest(input *ecs.ListTasksInput) ecs.ListTasksRequest {
	if *input.Cluster == "invalid-request-cluster" {
		return ecs.ListTasksRequest{
			Request: &aws.Request{
				Data:  &ecs.ListTasksOutput{},
				Error: errors.New("Error: List Tasks Request"),
			},
		}
	}
	return ecs.ListTasksRequest{
		Request: &aws.Request{
			Data: &m.ListTaskResp,
		},
	}
}
