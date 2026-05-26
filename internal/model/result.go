package model

type CheckStatus string

const (
	StatusPass    CheckStatus = "pass"
	StatusFail    CheckStatus = "fail"
	StatusSkipped CheckStatus = "skipped"
)

type CheckResult struct {
	Name   string
	Detail string
	Status CheckStatus
	Fix    string
	Link   string
}

type DiagnosisResult struct {
	Cluster    string
	Service    string
	Desired    int32
	Running    int32
	Pending    int32
	LaunchType string
	Checks     []CheckResult
	Cause      *CheckResult
	Healthy    bool
}
