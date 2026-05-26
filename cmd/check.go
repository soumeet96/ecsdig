package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	awsclient "github.com/soumeet96/ecsdig/internal/aws"
	"github.com/soumeet96/ecsdig/internal/checker"
	"github.com/soumeet96/ecsdig/internal/model"
	"github.com/soumeet96/ecsdig/internal/output"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check why an ECS service is not reaching desired count",
	Example: `  ecsdig check --cluster prod --service my-api
  ecsdig check --cluster prod --service my-api --region us-west-2
  ecsdig check --cluster prod --service my-api --output json
  ecsdig check --cluster local --service my-api --endpoint-url http://localhost:4566`,
	RunE: runCheck,
}

var opts model.Options

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringVar(&opts.Cluster, "cluster", "", "ECS cluster name or ARN (required)")
	checkCmd.Flags().StringVar(&opts.Service, "service", "", "ECS service name (required)")
	checkCmd.Flags().StringVar(&opts.Region, "region", "", "AWS region (default: from AWS config)")
	checkCmd.Flags().StringVar(&opts.Profile, "profile", "", "AWS profile name")
	checkCmd.Flags().StringVar(&opts.EndpointURL, "endpoint-url", "", "Override AWS endpoint URL (e.g. http://localhost:4566 for Floci)")
	checkCmd.Flags().StringVar(&opts.Output, "output", "table", "Output format: table or json")
	checkCmd.Flags().IntVar(&opts.LogLines, "log-lines", 10, "Number of CloudWatch log lines to show on crash")

	_ = checkCmd.MarkFlagRequired("cluster")
	_ = checkCmd.MarkFlagRequired("service")
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	clients, err := awsclient.NewClients(ctx, &opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing AWS clients: %v\n", err)
		os.Exit(2)
	}

	result, err := checker.Run(ctx, clients, &opts)
	if err != nil {
		var notFound *awsclient.ServiceNotFoundError
		if errors.As(err, &notFound) {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	switch opts.Output {
	case "json":
		output.PrintJSON(result)
	default:
		output.PrintTable(result)
	}

	if result.Healthy {
		os.Exit(0)
	}
	if result.Cause != nil {
		os.Exit(1)
	}
	os.Exit(1)
	return nil
}
