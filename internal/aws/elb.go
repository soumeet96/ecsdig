package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type TargetGroupInfo struct {
	ARN                 string
	Name                string
	HealthCheckPath     string
	HealthCheckPort     string
	HealthCheckProtocol string
	UnhealthyThreshold  int32
}

type TargetHealthInfo struct {
	ID          string
	Port        int32
	State       string
	Reason      string
	Description string
}

func GetTargetGroup(ctx context.Context, client *elasticloadbalancingv2.Client, arn string) (*TargetGroupInfo, error) {
	out, err := client.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{
		TargetGroupArns: []string{arn},
	})
	if err != nil {
		return nil, err
	}
	if len(out.TargetGroups) == 0 {
		return nil, nil
	}

	tg := out.TargetGroups[0]
	return &TargetGroupInfo{
		ARN:                 aws.ToString(tg.TargetGroupArn),
		Name:                aws.ToString(tg.TargetGroupName),
		HealthCheckPath:     aws.ToString(tg.HealthCheckPath),
		HealthCheckPort:     aws.ToString(tg.HealthCheckPort),
		HealthCheckProtocol: string(tg.HealthCheckProtocol),
		UnhealthyThreshold:  aws.ToInt32(tg.UnhealthyThresholdCount),
	}, nil
}

func GetTargetHealth(ctx context.Context, client *elasticloadbalancingv2.Client, targetGroupARN string) ([]TargetHealthInfo, error) {
	out, err := client.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(targetGroupARN),
	})
	if err != nil {
		return nil, err
	}

	var results []TargetHealthInfo
	for _, t := range out.TargetHealthDescriptions {
		th := TargetHealthInfo{
			State: string(t.TargetHealth.State),
		}
		if t.Target != nil {
			th.ID = aws.ToString(t.Target.Id)
			th.Port = aws.ToInt32(t.Target.Port)
		}
		if t.TargetHealth != nil {
			th.Reason = string(t.TargetHealth.Reason)
			th.Description = aws.ToString(t.TargetHealth.Description)
		}
		results = append(results, th)
	}
	return results, nil
}
