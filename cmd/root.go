package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/soumeet96/ecsdig/pkg/version"
)

var rootCmd = &cobra.Command{
	Use:   "ecsdig",
	Short: "Diagnose stuck ECS service deployments",
	Long: `ecsdig checks every possible reason why your ECS service is not
reaching its desired task count and tells you exactly what is wrong — and how to fix it.`,
	Version: version.Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
