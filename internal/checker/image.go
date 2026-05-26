package checker

import (
	"context"
	"fmt"

	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/model"
)

func checkImage(ctx context.Context, clients *awsclient.Clients, svc *awsclient.ServiceInfo, taskDef *awsclient.TaskDefinitionInfo, opts *model.Options) (*model.CheckResult, error) {
	cr := &model.CheckResult{Name: "Container Image"}

	for _, cd := range taskDef.Containers {
		img := awsclient.ParseECRImage(cd.Image)
		if img == nil {
			// not an ECR image (Docker Hub etc.) — skip
			continue
		}

		exists, err := awsclient.ImageExists(ctx, clients.ECR, img)
		if err != nil {
			return cr, fmt.Errorf("checking image %s: %w", cd.Image, err)
		}

		if !exists {
			cr.Status = model.StatusFail
			cr.Detail = fmt.Sprintf("image not found in ECR: %s:%s (repository: %s)", img.Repository, img.Tag, img.Repository)
			cr.Fix = fmt.Sprintf("push the image tag %q to ECR repository %q, or update the task definition to use an existing tag", img.Tag, img.Repository)
			cr.Link = fmt.Sprintf("https://console.aws.amazon.com/ecr/repositories/%s", img.Repository)
			return cr, nil
		}
	}

	cr.Status = model.StatusPass
	cr.Detail = "all ECR images verified"
	return cr, nil
}
