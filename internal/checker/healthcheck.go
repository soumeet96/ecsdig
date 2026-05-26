package checker

import (
	"context"
	"fmt"

	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/model"
)

func checkHealthCheck(ctx context.Context, clients *awsclient.Clients, svc *awsclient.ServiceInfo, taskDef *awsclient.TaskDefinitionInfo, opts *model.Options) (*model.CheckResult, error) {
	cr := &model.CheckResult{Name: "Health Check"}

	if len(svc.LoadBalancers) == 0 {
		cr.Status = model.StatusSkipped
		cr.Detail = "no load balancer attached to service"
		return cr, nil
	}

	for _, lb := range svc.LoadBalancers {
		tg, err := awsclient.GetTargetGroup(ctx, clients.ELB, lb.TargetGroupARN)
		if err != nil {
			return cr, err
		}

		targets, err := awsclient.GetTargetHealth(ctx, clients.ELB, lb.TargetGroupARN)
		if err != nil {
			return cr, err
		}

		var unhealthy []awsclient.TargetHealthInfo
		for _, t := range targets {
			if t.State == "unhealthy" || t.State == "unused" {
				unhealthy = append(unhealthy, t)
			}
		}

		if len(unhealthy) > 0 {
			detail := fmt.Sprintf("target group %s — health check path: %s\n", tg.Name, tg.HealthCheckPath)
			for _, t := range unhealthy {
				detail += fmt.Sprintf("  target %s:%d is %s — %s\n", t.ID, t.Port, t.State, t.Description)
			}

			cr.Status = model.StatusFail
			cr.Detail = detail
			cr.Fix = fmt.Sprintf(
				"health check %s %s is failing — verify the path returns HTTP 200 and the container listens on port %d",
				tg.HealthCheckProtocol, tg.HealthCheckPath, lb.ContainerPort,
			)
			cr.Link = fmt.Sprintf("https://console.aws.amazon.com/ec2/v2/home#TargetGroups:search=%s", tg.Name)
			return cr, nil
		}
	}

	cr.Status = model.StatusPass
	cr.Detail = "all targets healthy"
	return cr, nil
}
