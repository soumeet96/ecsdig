package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type ServiceInfo struct {
	Name              string
	ARN               string
	DesiredCount      int32
	RunningCount      int32
	PendingCount      int32
	TaskDefinitionARN string
	LoadBalancers     []LoadBalancerInfo
	Constraints       []ConstraintInfo
	LaunchType        string
	Events            []EventInfo
}

type LoadBalancerInfo struct {
	TargetGroupARN string
	ContainerName  string
	ContainerPort  int32
}

type ConstraintInfo struct {
	Type       string
	Expression string
}

type EventInfo struct {
	Message   string
	CreatedAt time.Time
}

type TaskInfo struct {
	ARN           string
	StopCode      string
	StoppedReason string
	Containers    []ContainerInfo
	StartedAt     *time.Time
	StoppedAt     *time.Time
}

type ContainerInfo struct {
	Name     string
	Image    string
	ExitCode *int32
	Reason   string
}

type TaskDefinitionInfo struct {
	ARN              string
	ExecutionRoleARN string
	CPU              string
	Memory           string
	Containers       []ContainerDef
}

type ContainerDef struct {
	Name            string
	Image           string
	CPU             int32
	Memory          int32
	LogGroup        string
	LogStreamPrefix string
	Essential       bool
}

type ContainerInstanceInfo struct {
	ARN             string
	EC2InstanceID   string
	Status          string
	RemainingCPU    int64
	RemainingMemory int64
	Attributes      []AttributeInfo
}

type AttributeInfo struct {
	Name  string
	Value string
}

func GetService(ctx context.Context, client *ecs.Client, cluster, service string) (*ServiceInfo, error) {
	out, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []string{service},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Services) == 0 {
		return nil, &ServiceNotFoundError{Cluster: cluster, Service: service}
	}

	svc := out.Services[0]
	info := &ServiceInfo{
		Name:              aws.ToString(svc.ServiceName),
		ARN:               aws.ToString(svc.ServiceArn),
		DesiredCount:      svc.DesiredCount,
		RunningCount:      svc.RunningCount,
		PendingCount:      svc.PendingCount,
		TaskDefinitionARN: aws.ToString(svc.TaskDefinition),
		LaunchType:        string(svc.LaunchType),
	}

	for _, lb := range svc.LoadBalancers {
		info.LoadBalancers = append(info.LoadBalancers, LoadBalancerInfo{
			TargetGroupARN: aws.ToString(lb.TargetGroupArn),
			ContainerName:  aws.ToString(lb.ContainerName),
			ContainerPort:  aws.ToInt32(lb.ContainerPort),
		})
	}

	for _, c := range svc.PlacementConstraints {
		info.Constraints = append(info.Constraints, ConstraintInfo{
			Type:       string(c.Type),
			Expression: aws.ToString(c.Expression),
		})
	}

	for i, e := range svc.Events {
		if i >= 5 {
			break
		}
		info.Events = append(info.Events, EventInfo{
			Message:   aws.ToString(e.Message),
			CreatedAt: aws.ToTime(e.CreatedAt),
		})
	}

	return info, nil
}

func GetStoppedTasks(ctx context.Context, client *ecs.Client, cluster, service string) ([]TaskInfo, error) {
	listOut, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		ServiceName:   aws.String(service),
		DesiredStatus: ecstypes.DesiredStatusStopped,
		MaxResults:    aws.Int32(5),
	})
	if err != nil {
		return nil, err
	}
	if len(listOut.TaskArns) == 0 {
		return nil, nil
	}

	descOut, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   listOut.TaskArns,
	})
	if err != nil {
		return nil, err
	}

	var tasks []TaskInfo
	for _, t := range descOut.Tasks {
		info := TaskInfo{
			ARN:           aws.ToString(t.TaskArn),
			StopCode:      string(t.StopCode),
			StoppedReason: aws.ToString(t.StoppedReason),
			StartedAt:     t.StartedAt,
			StoppedAt:     t.StoppedAt,
		}
		for _, c := range t.Containers {
			ci := ContainerInfo{
				Name:   aws.ToString(c.Name),
				Image:  aws.ToString(c.Image),
				Reason: aws.ToString(c.Reason),
			}
			if c.ExitCode != nil {
				ci.ExitCode = c.ExitCode
			}
			info.Containers = append(info.Containers, ci)
		}
		tasks = append(tasks, info)
	}
	return tasks, nil
}

func GetTaskDefinition(ctx context.Context, client *ecs.Client, taskDefARN string) (*TaskDefinitionInfo, error) {
	out, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefARN),
	})
	if err != nil {
		return nil, err
	}

	td := out.TaskDefinition
	info := &TaskDefinitionInfo{
		ARN:              aws.ToString(td.TaskDefinitionArn),
		ExecutionRoleARN: aws.ToString(td.ExecutionRoleArn),
		CPU:              aws.ToString(td.Cpu),
		Memory:           aws.ToString(td.Memory),
	}

	for _, cd := range td.ContainerDefinitions {
		def := ContainerDef{
			Name:      aws.ToString(cd.Name),
			Image:     aws.ToString(cd.Image),
			CPU:       cd.Cpu,
			Memory:    aws.ToInt32(cd.Memory),
			Essential: aws.ToBool(cd.Essential),
		}
		if cd.LogConfiguration != nil && string(cd.LogConfiguration.LogDriver) == "awslogs" {
			def.LogGroup = cd.LogConfiguration.Options["awslogs-group"]
			def.LogStreamPrefix = cd.LogConfiguration.Options["awslogs-stream-prefix"]
		}
		info.Containers = append(info.Containers, def)
	}

	return info, nil
}

func ListContainerInstances(ctx context.Context, client *ecs.Client, cluster string) ([]ContainerInstanceInfo, error) {
	listOut, err := client.ListContainerInstances(ctx, &ecs.ListContainerInstancesInput{
		Cluster: aws.String(cluster),
	})
	if err != nil {
		return nil, err
	}
	if len(listOut.ContainerInstanceArns) == 0 {
		return nil, nil
	}

	descOut, err := client.DescribeContainerInstances(ctx, &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(cluster),
		ContainerInstances: listOut.ContainerInstanceArns,
	})
	if err != nil {
		return nil, err
	}

	var instances []ContainerInstanceInfo
	for _, ci := range descOut.ContainerInstances {
		info := ContainerInstanceInfo{
			ARN:           aws.ToString(ci.ContainerInstanceArn),
			EC2InstanceID: aws.ToString(ci.Ec2InstanceId),
			Status:        aws.ToString(ci.Status),
		}
		for _, r := range ci.RemainingResources {
			switch aws.ToString(r.Name) {
			case "CPU":
				info.RemainingCPU = int64(r.IntegerValue)
			case "MEMORY":
				info.RemainingMemory = int64(r.IntegerValue)
			}
		}
		for _, a := range ci.Attributes {
			info.Attributes = append(info.Attributes, AttributeInfo{
				Name:  aws.ToString(a.Name),
				Value: aws.ToString(a.Value),
			})
		}
		instances = append(instances, info)
	}
	return instances, nil
}

type ServiceNotFoundError struct {
	Cluster string
	Service string
}

func (e *ServiceNotFoundError) Error() string {
	return "service " + e.Service + " not found in cluster " + e.Cluster
}
