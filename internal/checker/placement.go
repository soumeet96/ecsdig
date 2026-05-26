package checker

import (
	"context"
	"fmt"
	"strings"

	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/model"
)

func checkPlacement(ctx context.Context, clients *awsclient.Clients, svc *awsclient.ServiceInfo, taskDef *awsclient.TaskDefinitionInfo, opts *model.Options) (*model.CheckResult, error) {
	cr := &model.CheckResult{Name: "Placement Constraints"}

	if svc.LaunchType == "FARGATE" || svc.LaunchType == "" {
		cr.Status = model.StatusSkipped
		cr.Detail = "Fargate launch type — placement constraints not applicable"
		return cr, nil
	}

	if len(svc.Constraints) == 0 {
		cr.Status = model.StatusPass
		cr.Detail = "no placement constraints defined"
		return cr, nil
	}

	instances, err := awsclient.ListContainerInstances(ctx, clients.ECS, opts.Cluster)
	if err != nil {
		return cr, err
	}

	// for each memberOf constraint, check if any instance satisfies it
	for _, constraint := range svc.Constraints {
		if constraint.Type != "memberOf" || constraint.Expression == "" {
			continue
		}

		matched := anyInstanceMatchesAttribute(instances, constraint.Expression)
		if !matched {
			cr.Status = model.StatusFail
			cr.Detail = fmt.Sprintf("placement constraint %q cannot be satisfied by any instance in the cluster", constraint.Expression)
			cr.Fix = fmt.Sprintf("add the required attribute to an EC2 instance in cluster %q, or remove/update the placement constraint", opts.Cluster)
			cr.Link = fmt.Sprintf("https://console.aws.amazon.com/ecs/home#/clusters/%s/containerInstances", opts.Cluster)
			return cr, nil
		}
	}

	cr.Status = model.StatusPass
	cr.Detail = "all placement constraints can be satisfied"
	return cr, nil
}

// anyInstanceMatchesAttribute does a simple substring check on the expression
// against instance attribute names and values. This covers the common case of
// attribute:ecs.instance-type == t3.medium style expressions.
func anyInstanceMatchesAttribute(instances []awsclient.ContainerInstanceInfo, expression string) bool {
	expr := strings.ToLower(expression)
	for _, inst := range instances {
		for _, attr := range inst.Attributes {
			attrStr := strings.ToLower(attr.Name + "==" + attr.Value)
			if strings.Contains(expr, strings.ToLower(attr.Name)) &&
				(attr.Value == "" || strings.Contains(expr, strings.ToLower(attr.Value))) {
				_ = attrStr
				return true
			}
		}
	}
	return false
}
