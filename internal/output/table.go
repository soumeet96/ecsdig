package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/soumeet96/ecsdig/internal/model"
)

const (
	colCheck   = 22
	colDetail  = 52
	colVerdict = 12
)

var (
	green  = color.New(color.FgGreen, color.Bold)
	red    = color.New(color.FgRed, color.Bold)
	yellow = color.New(color.FgYellow, color.Bold)
	bold   = color.New(color.Bold)
	grey   = color.New(color.FgHiBlack)
)

func PrintTable(result *model.DiagnosisResult) {
	fmt.Fprintln(os.Stdout)
	bold.Fprintf(os.Stdout, "  Checking %s", result.Cluster)
	fmt.Fprintf(os.Stdout, "  ·  service %s  ·  desired %d  ·  running %d\n\n",
		result.Service, result.Desired, result.Running)

	// header
	bold.Fprintf(os.Stdout, "  %-*s  %-*s  %s\n", colCheck, "CHECK", colDetail, "DETAIL", "VERDICT")
	grey.Fprintf(os.Stdout, "  %s\n", strings.Repeat("─", colCheck+colDetail+colVerdict+4))

	for _, cr := range result.Checks {
		detail := truncate(strings.SplitN(cr.Detail, "\n", 2)[0], colDetail)
		fmt.Fprintf(os.Stdout, "  %-*s  %-*s  ", colCheck, cr.Name, colDetail, detail)
		verdictColor(cr.Status)
	}

	fmt.Fprintln(os.Stdout)

	if result.Healthy {
		green.Fprintf(os.Stdout, "  RESULT:  ✓  Service is healthy — desired %d == running %d\n\n",
			result.Desired, result.Running)
		return
	}

	if result.Cause != nil {
		red.Fprintf(os.Stdout, "  RESULT:  ✗  Blocked at: %s\n\n", result.Cause.Name)

		for _, line := range strings.Split(strings.TrimRight(result.Cause.Detail, "\n"), "\n") {
			fmt.Fprintf(os.Stdout, "  %s\n", line)
		}
		fmt.Fprintln(os.Stdout)

		if result.Cause.Fix != "" {
			fmt.Fprintf(os.Stdout, "  FIX:     %s\n", result.Cause.Fix)
		}
		if result.Cause.Link != "" {
			grey.Fprintf(os.Stdout, "  LINK:    %s\n", result.Cause.Link)
		}
	} else {
		yellow.Fprintf(os.Stdout, "  RESULT:  ?  Could not determine cause — check service events in the AWS console\n")
	}

	fmt.Fprintln(os.Stdout)
}

func verdictColor(status model.CheckStatus) {
	switch status {
	case model.StatusPass:
		green.Fprintln(os.Stdout, "✓  ok")
	case model.StatusFail:
		red.Fprintln(os.Stdout, "✗  BLOCKED")
	case model.StatusSkipped:
		grey.Fprintln(os.Stdout, "–  skipped")
	default:
		fmt.Fprintln(os.Stdout, "?")
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
