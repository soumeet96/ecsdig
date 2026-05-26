# ecsdig

> **Why is my ECS service stuck?** — Find out in one command.

`ecsdig` is an open-source CLI tool that diagnoses why your AWS ECS service is not reaching its desired task count. Instead of manually checking stopped task logs, ALB health checks, ECR images, IAM roles, and cluster capacity across multiple AWS console pages, you run one command and get a layer-by-layer verdict telling you exactly where the problem is — and how to fix it.

No SSH. No exec into containers. No instance access required.

---

## The Problem

When your ECS service shows `desired: 3, running: 1`, the AWS console tells you:

```
service my-api was unable to place a task for 5 minutes
```

That's it. Nothing else. To find the actual cause you have to manually check:

```
Stopped tasks + exit codes     →  is the container crashing on startup?
CloudWatch Logs                →  what was the last log line before it died?
ALB target group health        →  is the health check path wrong?
ECR image                      →  does the image tag actually exist?
IAM execution role             →  does it have the right permissions?
Cluster capacity               →  is there enough CPU / memory?
Placement constraints          →  can any instance satisfy them?
```

That is 6–7 checks across 4–5 different AWS console pages. **ecsdig does all of this in one command, in seconds.**

---

## Demo

```
$ ecsdig check --cluster prod --service my-api

  Checking prod  ·  service my-api  ·  desired 3  ·  running 1

  CHECK                   DETAIL                                                VERDICT
  ──────────────────────────────────────────────────────────────────────────────────────────
  Stopped Tasks           container "my-app" exited with code 1                ✗  BLOCKED
                          Last log lines:
                            [ERROR] DB_HOST env var not set
                            [FATAL] failed to connect to database

  RESULT:  ✗  Blocked at: Stopped Tasks

  container "my-app" is crashing on startup — check application logs and fix the exit code 1 error

  FIX:     container "my-app" is crashing on startup — check application logs and fix the exit code 1 error
  LINK:    https://console.aws.amazon.com/cloudwatch/home#logsV2:log-groups
```

```
$ ecsdig check --cluster prod --service my-api

  Checking prod  ·  service my-api  ·  desired 3  ·  running 3

  RESULT:  ✓  Service is healthy — desired 3 == running 3
```

---

## How It Works

`ecsdig` evaluates every possible failure reason in order, stopping at the first confirmed cause:

```
ECS Service (desired != running)
        │
        ▼
Check 1 — Stopped Tasks       Are recent tasks crashing? What exit code? Last log lines?
        │
        ▼
Check 2 — Health Check        Is the ALB health check failing? Which targets? Which path?
        │
        ▼
Check 3 — Container Image     Does the image tag exist in ECR?
        │
        ▼
Check 4 — IAM Execution Role  Does the role have AmazonECSTaskExecutionRolePolicy?
        │
        ▼
Check 5 — Cluster Capacity    Is there enough CPU / memory? (EC2 launch type only)
        │
        ▼
Check 6 — Placement           Can any instance satisfy the placement constraints? (EC2 only)
        │
        ▼
Verdict + Fix + Console Link
```

---

## Installation

### Using Go

```bash
go install github.com/soumeet96/ecsdig@latest
```

> Make sure Go's bin directory is in your PATH:
> ```bash
> export PATH="$PATH:$(go env GOPATH)/bin"
> ```
> Add this to your `~/.zshrc` or `~/.bashrc` to make it permanent.

### Download Binary

Download the latest binary from the [Releases](https://github.com/soumeet96/ecsdig/releases) page.

```bash
# macOS (Apple Silicon)
curl -L https://github.com/soumeet96/ecsdig/releases/download/v0.1.0/ecsdig_darwin_arm64.tar.gz | tar xz
sudo mv ecsdig /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/soumeet96/ecsdig/releases/download/v0.1.0/ecsdig_linux_amd64.tar.gz | tar xz
sudo mv ecsdig /usr/local/bin/
```

---

## AWS Credentials

`ecsdig` uses your existing AWS credentials. All standard credential sources are supported:

```bash
# Option 1 — AWS CLI profile
ecsdig check --cluster prod --service my-api --profile myprofile

# Option 2 — Environment variables
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_DEFAULT_REGION=us-east-1
ecsdig check --cluster prod --service my-api

# Option 3 — IAM role (EC2 instance role, ECS task role, etc.)
ecsdig check --cluster prod --service my-api
```

### Required IAM Permissions

`ecsdig` is fully read-only. It only needs `Describe*` and `List*` permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:DescribeServices",
        "ecs:DescribeTasks",
        "ecs:ListTasks",
        "ecs:DescribeTaskDefinition",
        "ecs:DescribeContainerInstances",
        "ecs:ListContainerInstances",
        "logs:DescribeLogStreams",
        "logs:GetLogEvents",
        "elasticloadbalancing:DescribeTargetGroups",
        "elasticloadbalancing:DescribeTargetHealth",
        "ecr:DescribeImages",
        "iam:ListAttachedRolePolicies"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## Commands

### `ecsdig check` — Diagnose a stuck service

```bash
ecsdig check --cluster <cluster-name> --service <service-name> [flags]
```

| Flag | Description | Default |
|---|---|---|
| `--cluster` | ECS cluster name or ARN **(required)** | — |
| `--service` | ECS service name **(required)** | — |
| `--region` | AWS region | From AWS config |
| `--profile` | AWS profile name | From AWS config |
| `--output` | Output format: `table` or `json` | `table` |
| `--log-lines` | Number of CloudWatch log lines to show on crash | `10` |
| `--endpoint-url` | Override AWS endpoint (e.g. for local testing with Floci) | — |

**Examples:**

```bash
# Basic check
ecsdig check --cluster prod --service my-api

# Different region and profile
ecsdig check --cluster prod --service my-api --region ap-south-1 --profile prod

# JSON output for scripting
ecsdig check --cluster prod --service my-api --output json

# Show more log lines on crash
ecsdig check --cluster prod --service my-api --log-lines 25
```

---

## Exit Codes

Use `ecsdig` in CI/CD pipelines and deployment scripts:

| Code | Meaning |
|---|---|
| `0` | Service is healthy — desired == running |
| `1` | Problem found — service is stuck |
| `2` | Error — cluster/service not found, missing permissions, etc. |

```bash
ecsdig check --cluster prod --service my-api
if [ $? -eq 1 ]; then
  echo "Deployment stuck — investigate before proceeding"
  exit 1
fi
```

---

## Local Testing with Floci

You can test `ecsdig` locally without a real AWS account using [Floci](https://floci.io) — a free, open-source AWS emulator.

```bash
# start Floci
docker compose up -d

# point AWS CLI at Floci
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_DEFAULT_REGION=us-east-1
export AWS_ACCESS_KEY_ID=fake
export AWS_SECRET_ACCESS_KEY=fake

# set up a broken ECS service
aws ecs create-cluster --cluster-name test-cluster
aws ecs register-task-definition --family my-task \
  --container-definitions '[{"name":"app","image":"000000000000.dkr.ecr.us-east-1.amazonaws.com/my-repo:missing","cpu":256,"memory":512,"essential":true}]'
aws ecs create-service --cluster test-cluster --service-name my-api --task-definition my-task --desired-count 3

# run ecsdig against it
ecsdig check --cluster test-cluster --service my-api
```

---

## Why Not Use AWS Tools?

| | AWS Console | CloudWatch | ecsdig |
|---|---|---|---|
| **Single command** | ✗ | ✗ | ✓ |
| **Shows root cause** | ✗ | Partial | ✓ |
| **Actionable fix** | ✗ | ✗ | ✓ |
| **CI/CD friendly** | ✗ | ✗ | ✓ (exit codes) |
| **Free** | ✓ | ✓ | ✓ |
| **Terminal native** | ✗ | ✗ | ✓ |

---

## Roadmap

- [x] Stopped task crash detection with log lines
- [x] ALB health check diagnosis
- [x] ECR image existence check
- [x] IAM execution role check
- [x] Cluster capacity check (EC2)
- [x] Placement constraint check (EC2)
- [ ] Fargate capacity unavailability detection
- [ ] Secrets Manager / SSM access failure detection
- [ ] JSON output for CI/CD integration
- [ ] Multi-service check (`--all-services`)
- [ ] Slack / webhook notification on failure

---

## Contributing

Contributions are welcome. Please open an issue before submitting a large PR.

1. Fork the repository
2. Create a feature branch from `main`
   ```bash
   git checkout -b feat/your-feature-name
   ```
3. Make your changes and run `make check`
4. Push and open a PR targeting `main`

---

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.
