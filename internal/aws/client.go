package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/soumeet96/ecsdig/internal/model"
)

type Clients struct {
	ECS  *ecs.Client
	Logs *cloudwatchlogs.Client
	ELB  *elasticloadbalancingv2.Client
	ECR  *ecr.Client
	IAM  *iam.Client
}

func NewClients(ctx context.Context, opts *model.Options) (*Clients, error) {
	var cfgOpts []func(*config.LoadOptions) error

	if opts.Region != "" {
		cfgOpts = append(cfgOpts, config.WithRegion(opts.Region))
	}
	if opts.Profile != "" {
		cfgOpts = append(cfgOpts, config.WithSharedConfigProfile(opts.Profile))
	}
	if opts.EndpointURL != "" {
		// override all service endpoints — used for Floci / LocalStack
		cfgOpts = append(cfgOpts, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{URL: opts.EndpointURL, HostnameImmutable: true}, nil
				},
			),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, err
	}

	return &Clients{
		ECS:  ecs.NewFromConfig(cfg),
		Logs: cloudwatchlogs.NewFromConfig(cfg),
		ELB:  elasticloadbalancingv2.NewFromConfig(cfg),
		ECR:  ecr.NewFromConfig(cfg),
		IAM:  iam.NewFromConfig(cfg),
	}, nil
}
