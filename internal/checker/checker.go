package checker

import (
	"context"
	"fmt"

	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/model"
)

func Run(ctx context.Context, clients *awsclient.Clients, opts *model.Options) (*model.DiagnosisResult, error) {
	svc, err := awsclient.GetService(ctx, clients.ECS, opts.Cluster, opts.Service)
	if err != nil {
		return nil, err
	}

	taskDef, err := awsclient.GetTaskDefinition(ctx, clients.ECS, svc.TaskDefinitionARN)
	if err != nil {
		return nil, fmt.Errorf("describing task definition: %w", err)
	}

	result := &model.DiagnosisResult{
		Cluster:    opts.Cluster,
		Service:    opts.Service,
		Desired:    svc.DesiredCount,
		Running:    svc.RunningCount,
		Pending:    svc.PendingCount,
		LaunchType: svc.LaunchType,
	}

	if svc.DesiredCount == svc.RunningCount && svc.RunningCount > 0 {
		result.Healthy = true
		return result, nil
	}

	type checkFn func(context.Context, *awsclient.Clients, *awsclient.ServiceInfo, *awsclient.TaskDefinitionInfo, *model.Options) (*model.CheckResult, error)

	checks := []checkFn{
		checkStoppedTasks,
		checkHealthCheck,
		checkImage,
		checkIAM,
		checkCapacity,
		checkPlacement,
	}

	for _, fn := range checks {
		cr, err := fn(ctx, clients, svc, taskDef, opts)
		if err != nil {
			// non-fatal: record as skipped and continue
			result.Checks = append(result.Checks, model.CheckResult{
				Name:   cr.Name,
				Detail: "skipped: " + err.Error(),
				Status: model.StatusSkipped,
			})
			continue
		}
		result.Checks = append(result.Checks, *cr)
		if cr.Status == model.StatusFail {
			result.Cause = cr
			return result, nil
		}
	}

	return result, nil
}
