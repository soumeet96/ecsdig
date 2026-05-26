package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

var requiredExecutionRolePolicies = []string{
	"ecr:GetAuthorizationToken",
	"ecr:BatchCheckLayerAvailability",
	"ecr:GetDownloadUrlForLayer",
	"ecr:BatchGetImage",
	"logs:CreateLogStream",
	"logs:PutLogEvents",
}

type RoleInfo struct {
	Name             string
	ARN              string
	AttachedPolicies []string
	HasExecutionRole bool
}

func GetRoleInfo(ctx context.Context, client *iam.Client, roleARN string) (*RoleInfo, error) {
	// extract role name from ARN: arn:aws:iam::123:role/my-role
	parts := strings.Split(roleARN, "/")
	roleName := parts[len(parts)-1]

	out, err := client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, err
	}

	info := &RoleInfo{
		Name: roleName,
		ARN:  roleARN,
	}

	for _, p := range out.AttachedPolicies {
		policyName := aws.ToString(p.PolicyName)
		info.AttachedPolicies = append(info.AttachedPolicies, policyName)
		if policyName == "AmazonECSTaskExecutionRolePolicy" {
			info.HasExecutionRole = true
		}
	}

	return info, nil
}
