package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type ECRImage struct {
	Registry   string
	Repository string
	Tag        string
}

// ParseECRImage parses an ECR image URI into its components.
// Returns nil if the URI is not an ECR image (e.g. Docker Hub).
func ParseECRImage(imageURI string) *ECRImage {
	if !strings.Contains(imageURI, ".dkr.ecr.") {
		return nil
	}

	parts := strings.SplitN(imageURI, "/", 2)
	if len(parts) != 2 {
		return nil
	}

	registry := parts[0]
	repoAndTag := parts[1]

	repo, tag, _ := strings.Cut(repoAndTag, ":")
	if tag == "" {
		tag = "latest"
	}

	return &ECRImage{
		Registry:   registry,
		Repository: repo,
		Tag:        tag,
	}
}

func ImageExists(ctx context.Context, client *ecr.Client, image *ECRImage) (bool, error) {
	_, err := client.DescribeImages(ctx, &ecr.DescribeImagesInput{
		RepositoryName: aws.String(image.Repository),
		ImageIds: []ecrtypes.ImageIdentifier{
			{ImageTag: aws.String(image.Tag)},
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "ImageNotFoundException") || strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
