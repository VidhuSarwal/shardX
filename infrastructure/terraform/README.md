# ShardX Terraform (Phase 1)

Minimal, hackathon-cost-conscious infrastructure-as-code for ShardX, an
AWS-native file storage platform. This is Phase 1 of a larger migration --
scope is intentionally narrow: budget guardrail, S3 storage, identity/IAM,
event orchestration scaffolding, observability, and a standalone search
domain. DynamoDB tables are explicitly deferred (see
`modules/storage/main.tf` TODO) to a later task once the Go DynamoDB
implementation lands.

## Directory layout

```
infrastructure/terraform/
├── main.tf              # root module: budget, storage, identity, orchestration, observability
├── variables.tf          # root-level variables (region, project name, budget email/threshold, ...)
├── outputs.tf            # key ARNs/names surfaced from the root module
├── providers.tf           # AWS provider ~> 5.0, region from variable
├── modules/
│   ├── budget/            # AWS Budget + SNS email alerts (80%/100%)
│   ├── storage/           # S3 shards bucket + KMS CMK
│   ├── identity/          # Cognito User Pool/Client + 3 IAM roles
│   ├── orchestration/     # EventBridge bus + SQS/DLQ + Step Functions placeholder
│   ├── observability/     # CloudWatch Log Group + CloudTrail + alarms
│   └── search/            # OpenSearch domain (instance-based) -- reusable module
└── search/                # STANDALONE root, own state, wraps modules/search
    ├── main.tf
    ├── providers.tf
    ├── variables.tf
    ├── outputs.tf
    └── terraform.tfvars.example
```

**Why `search/` is separate:** the OpenSearch `t3.small.search` domain is
the only resource in this project with meaningful per-hour billing. Keeping
it in its own root with its own state means you can `terraform apply` it
only while actively developing against it, and `terraform destroy` it
between sessions, without touching (or risking) anything else. It is
NOT referenced by the root `main.tf` and is not part of the default apply
path.

## Prerequisites

All commands below assume you have an AWS CLI profile named `shardx`
configured locally (`aws configure --profile shardx` or equivalent SSO
setup). Export it before running any terraform command in either root:

```sh
export AWS_PROFILE=shardx
```

No credentials or profile names are hardcoded anywhere in this repo --
`providers.tf` only reads `var.aws_region`; auth comes entirely from your
shell environment / AWS CLI profile chain.

## Apply order / runbook

### 1. Budget module -- apply FIRST, standalone

The budget module is a safety net and should exist before any billable
resource is created. It lives inside the root module (not a separate
directory) but should be targeted on its own for the very first apply:

```sh
cd infrastructure/terraform
export AWS_PROFILE=shardx
terraform init
terraform apply -target=module.budget \
  -var="budget_alert_email=you@example.com"
```

Confirm the SNS email subscription (check your inbox and click "Confirm
subscription") before moving on, or you won't receive alerts.

### 2. Root module -- the "leave running" resources

Once the budget guardrail is confirmed, apply the rest of the root module.
These resources are cheap enough to leave running for the full month:

```sh
terraform apply \
  -var="budget_alert_email=you@example.com"
```

This provisions, in order: storage (S3 + KMS), identity (Cognito + IAM
roles), orchestration (EventBridge + SQS/DLQ + Step Functions), and
observability (CloudWatch + CloudTrail).

To tear all of this down later:

```sh
terraform destroy -var="budget_alert_email=you@example.com"
```

### 3. Search module -- apply ONLY during active development sessions

The OpenSearch domain lives in `search/` with independent state. Copy the
role ARNs you need from the root module's outputs first:

```sh
cd infrastructure/terraform
terraform output api_role_arn shard_worker_role_arn
```

Then, in the `search/` directory:

```sh
cd infrastructure/terraform/search
export AWS_PROFILE=shardx
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars, paste in the role ARNs from above
terraform init
terraform apply
```

**When you're done for the session, destroy it:**

```sh
terraform destroy
```

Re-apply it the next time you need search. Because it's a separate state
file, this never touches the S3 bucket, Cognito pool, or anything else in
the main root.

## Estimated costs (us-east-1, approximate)

Resources from the root module (`main.tf`), if left running all month:

| Resource | Approx. cost |
|---|---|
| S3 bucket (shards) | ~$0.023/GB-month + request costs; near-zero at hackathon scale |
| KMS CMK | $1/month + $0.03 per 10k requests |
| Cognito User Pool | Free tier covers most hackathon usage (50k MAUs free) |
| SQS (queue + DLQ) | Free tier covers 1M requests/month; near-zero |
| EventBridge custom bus | $1 per million events published; near-zero |
| Step Functions (placeholder, Pass states only) | Free tier covers 4k transitions/month; near-zero |
| CloudWatch Logs | ~$0.50/GB ingested + $0.03/GB stored; small at this scale |
| CloudTrail (management events, single trail) | First trail's management events are free; S3 storage for logs is a few cents |
| AWS Budgets | First 2 budgets are free |
| **Total, left running all month** | **roughly $5-10/month** |

Resource from the separate `search/` root:

| Resource | Approx. cost |
|---|---|
| OpenSearch `t3.small.search` (single node, 10GB gp3 EBS) | ~$0.036/hr instance + ~$0.10/GB-month EBS -- roughly **$26/month if left running continuously**, but designed to be destroyed between sessions |

**Recommendation:** only run `terraform apply` in `search/` while you're
actively testing search functionality, and `terraform destroy` it
immediately after. A few hours per day of usage keeps this well under a
dollar.

## Assumptions and simplifications (Phase 1 scope)

- **`shards_bucket_name` (default `shardx-prod-shards`) must be changed
  before your first real apply.** S3 bucket names are globally unique
  across *all* AWS accounts, not just yours -- the default will very likely
  already be taken. Override it with something like
  `shardx-prod-shards-<your-account-id>`.
- No remote backend (S3 + DynamoDB lock table) is configured; state is
  local. Add a `backend "s3"` block once a shared backend is provisioned --
  intentionally out of scope for Phase 1.
- Cognito has no custom domain / hosted UI; it's just enough for
  `InitiateAuth`-based JWT issuance (`ALLOW_USER_PASSWORD_AUTH` /
  `ALLOW_USER_SRP_AUTH` + refresh token flows).
- IAM role trust policies allow both `lambda.amazonaws.com` and
  `ecs-tasks.amazonaws.com` as assumed-role principals, since the actual
  compute platform (Lambda vs. ECS/Fargate) hasn't been finalized yet in
  this phase.
- The Step Functions state machine uses `Pass` states for every lifecycle
  step (`VALIDATE -> FRAGMENTED -> UPLOAD_SHARDS -> VERIFY_CHECKSUMS ->
  REGISTER_METADATA -> HEALTHY`). Real task ARNs (Lambda invokes, ECS
  RunTask, etc.) are wired in a later phase; see the comment in
  `modules/orchestration/step_functions.tf`.
- `security-worker-role` only has CloudWatch Logs write access today. It's
  a placeholder reserved for a future phase that wires up GuardDuty/Macie
  finding processing.
- DynamoDB tables are NOT defined anywhere in this Terraform. See the
  one-line TODO in `modules/storage/main.tf` -- they're deferred to when
  the Go DynamoDB implementation lands.
- The OpenSearch domain uses IAM-based (SigV4) access control restricted to
  specific role ARNs, not Cognito-based dashboards auth or fine-grained
  access control (which would add complexity/cost not needed yet).
- CloudTrail is single-region (matches `var.aws_region`) and covers
  management events only; data events (e.g., S3 object-level) are out of
  scope for Phase 1 cost reasons.

## Validation note

`terraform fmt -recursive` and `terraform validate` (against `terraform init
-backend=false`, no AWS credentials involved) were both run successfully
against this configuration -- in both the root module
(`infrastructure/terraform/`) and the standalone `search/` root -- using
Terraform v1.16.3 and the `hashicorp/aws` ~> 5.0 provider. This confirms the
HCL is syntactically valid and internally consistent (variable references,
module wiring, resource argument names), but it does NOT confirm the
configuration will apply cleanly against a real AWS account -- `terraform
plan`/`apply` were not run since no AWS credentials are available in this
environment. Review the plan carefully on your first real `apply`.
