package checker

import (
	"context"
	"fmt"
	"strconv"

	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/model"
)

func checkCapacity(ctx context.Context, clients *awsclient.Clients, svc *awsclient.ServiceInfo, taskDef *awsclient.TaskDefinitionInfo, opts *model.Options) (*model.CheckResult, error) {
	cr := &model.CheckResult{Name: "Cluster Capacity"}

	// Fargate manages its own capacity — this check only applies to EC2
	if svc.LaunchType == "FARGATE" || svc.LaunchType == "" {
		cr.Status = model.StatusSkipped
		cr.Detail = "Fargate launch type — capacity managed by AWS"
		return cr, nil
	}

	instances, err := awsclient.ListContainerInstances(ctx, clients.ECS, opts.Cluster)
	if err != nil {
		return cr, err
	}
	if len(instances) == 0 {
		cr.Status = model.StatusFail
		cr.Detail = "no container instances registered in cluster"
		cr.Fix = "add EC2 instances to the ECS cluster or switch to Fargate launch type"
		cr.Link = fmt.Sprintf("https://console.aws.amazon.com/ecs/home#/clusters/%s/containerInstances", opts.Cluster)
		return cr, nil
	}

	// determine task resource requirements
	requiredCPU, requiredMem := taskResourceRequirements(taskDef)

	var canFit bool
	for _, inst := range instances {
		if inst.Status != "ACTIVE" {
			continue
		}
		if inst.RemainingCPU >= requiredCPU && inst.RemainingMemory >= requiredMem {
			canFit = true
			break
		}
	}

	if !canFit {
		maxCPU, maxMem := maxAvailable(instances)
		cr.Status = model.StatusFail
		cr.Detail = fmt.Sprintf(
			"task needs %d CPU units and %d MiB — max available on any instance: %d CPU / %d MiB",
			requiredCPU, requiredMem, maxCPU, maxMem,
		)
		cr.Fix = "scale up the EC2 instances in the cluster, reduce the task CPU/memory reservation, or switch to Fargate"
		cr.Link = fmt.Sprintf("https://console.aws.amazon.com/ecs/home#/clusters/%s", opts.Cluster)
		return cr, nil
	}

	cr.Status = model.StatusPass
	cr.Detail = fmt.Sprintf("%d instance(s) with sufficient capacity", len(instances))
	return cr, nil
}

func taskResourceRequirements(taskDef *awsclient.TaskDefinitionInfo) (cpu, mem int64) {
	// Fargate-style: top-level cpu/memory strings
	if taskDef.CPU != "" {
		if v, err := strconv.ParseInt(taskDef.CPU, 10, 64); err == nil {
			cpu = v
		}
	}
	if taskDef.Memory != "" {
		if v, err := strconv.ParseInt(taskDef.Memory, 10, 64); err == nil {
			mem = v
		}
	}
	if cpu > 0 && mem > 0 {
		return
	}
	// EC2-style: sum container-level resources
	for _, cd := range taskDef.Containers {
		cpu += int64(cd.CPU)
		if cd.Memory > 0 {
			mem += int64(cd.Memory)
		}
	}
	return
}

func maxAvailable(instances []awsclient.ContainerInstanceInfo) (cpu, mem int64) {
	for _, inst := range instances {
		if inst.RemainingCPU > cpu {
			cpu = inst.RemainingCPU
		}
		if inst.RemainingMemory > mem {
			mem = inst.RemainingMemory
		}
	}
	return
}
