package checker

import (
	"context"
	"fmt"
	"strings"

	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/model"
)

func checkStoppedTasks(ctx context.Context, clients *awsclient.Clients, svc *awsclient.ServiceInfo, taskDef *awsclient.TaskDefinitionInfo, opts *model.Options) (*model.CheckResult, error) {
	cr := &model.CheckResult{Name: "Stopped Tasks"}

	tasks, err := awsclient.GetStoppedTasks(ctx, clients.ECS, opts.Cluster, opts.Service)
	if err != nil {
		return cr, err
	}
	if len(tasks) == 0 {
		cr.Status = model.StatusPass
		cr.Detail = "no recently stopped tasks"
		return cr, nil
	}

	// look for tasks that crashed (non-zero exit code or essential container exited)
	for _, t := range tasks {
		for _, c := range t.Containers {
			if c.ExitCode != nil && *c.ExitCode != 0 {
				detail := fmt.Sprintf("container %q exited with code %d", c.Name, *c.ExitCode)

				// try to fetch last log lines
				logLines := fetchLogs(ctx, clients, taskDef, c.Name, opts.LogLines)
				if len(logLines) > 0 {
					detail += "\n  Last log lines:\n"
					for _, l := range logLines {
						detail += "    " + strings.TrimSpace(l.Message) + "\n"
					}
				}

				cr.Status = model.StatusFail
				cr.Detail = detail
				cr.Fix = fmt.Sprintf("container %q is crashing on startup — check application logs and fix the exit code %d error", c.Name, *c.ExitCode)
				cr.Link = fmt.Sprintf("https://console.aws.amazon.com/cloudwatch/home#logsV2:log-groups")
				return cr, nil
			}
		}

		// task stopped but containers show no exit code — check stop reason
		if t.StopCode == "EssentialTaskExited" && t.StoppedReason != "" {
			cr.Status = model.StatusFail
			cr.Detail = "task stopped: " + t.StoppedReason
			cr.Fix = "investigate why the essential container exited — check CloudWatch Logs for the task"
			return cr, nil
		}

		if t.StopCode == "CannotPullContainerError" {
			cr.Status = model.StatusFail
			cr.Detail = "container image pull failed: " + t.StoppedReason
			cr.Fix = "check that the image exists in ECR and the task execution role has ecr:GetAuthorizationToken permission"
			return cr, nil
		}
	}

	cr.Status = model.StatusPass
	cr.Detail = fmt.Sprintf("%d stopped task(s) found, none with crash exit codes", len(tasks))
	return cr, nil
}

func fetchLogs(ctx context.Context, clients *awsclient.Clients, taskDef *awsclient.TaskDefinitionInfo, containerName string, n int) []awsclient.LogLine {
	if n <= 0 {
		n = 10
	}
	for _, cd := range taskDef.Containers {
		if cd.Name == containerName && cd.LogGroup != "" {
			lines, _ := awsclient.GetLastLogLines(ctx, clients.Logs, cd.LogGroup, cd.LogStreamPrefix, containerName, n)
			return lines
		}
	}
	return nil
}
