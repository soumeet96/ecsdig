package checker

import (
	"context"
	"fmt"
	"strings"

	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/model"
)

func checkIAM(ctx context.Context, clients *awsclient.Clients, svc *awsclient.ServiceInfo, taskDef *awsclient.TaskDefinitionInfo, opts *model.Options) (*model.CheckResult, error) {
	cr := &model.CheckResult{Name: "IAM Execution Role"}

	if taskDef.ExecutionRoleARN == "" {
		cr.Status = model.StatusFail
		cr.Detail = "no execution role set on task definition"
		cr.Fix = "add an execution role with AmazonECSTaskExecutionRolePolicy to the task definition"
		cr.Link = fmt.Sprintf("https://console.aws.amazon.com/ecs/home#/taskDefinitions/%s", taskDefName(taskDef.ARN))
		return cr, nil
	}

	role, err := awsclient.GetRoleInfo(ctx, clients.IAM, taskDef.ExecutionRoleARN)
	if err != nil {
		return cr, err
	}

	if !role.HasExecutionRole {
		cr.Status = model.StatusFail
		cr.Detail = fmt.Sprintf("role %q does not have AmazonECSTaskExecutionRolePolicy attached\n  attached policies: %s",
			role.Name, strings.Join(role.AttachedPolicies, ", "))
		cr.Fix = fmt.Sprintf("attach AmazonECSTaskExecutionRolePolicy to IAM role %q", role.Name)
		cr.Link = fmt.Sprintf("https://console.aws.amazon.com/iam/home#/roles/%s", role.Name)
		return cr, nil
	}

	cr.Status = model.StatusPass
	cr.Detail = fmt.Sprintf("role %q has AmazonECSTaskExecutionRolePolicy", role.Name)
	return cr, nil
}

func taskDefName(arn string) string {
	parts := strings.Split(arn, "/")
	return parts[len(parts)-1]
}
