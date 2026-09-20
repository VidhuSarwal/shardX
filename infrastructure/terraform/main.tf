# ShardX Phase 1 root module.
#
# Module apply order (see README.md for the full runbook):
#   1. budget        -- safety net, apply first and standalone
#                        (terraform apply -target=module.budget)
#   2. storage       -- S3 bucket + KMS CMK
#   3. orchestration -- EventBridge + SQS + Step Functions
#   4. identity      -- Cognito + IAM roles (consumes storage + orchestration outputs)
#   5. observability -- CloudWatch + CloudTrail (consumes budget's SNS topic)
#
# The `search` module (OpenSearch) is intentionally NOT wired in here. It
# lives in its own root at infrastructure/terraform/search/ with independent
# state, so it can be applied/destroyed on its own without touching anything
# below. This keeps the only meaningfully-hourly-billed resource easy to
# tear down between development sessions.

module "budget" {
  source = "./modules/budget"

  project_name       = var.project_name
  monthly_budget_usd = var.monthly_budget_usd
  alert_email        = var.budget_alert_email
}

module "storage" {
  source = "./modules/storage"

  project_name = var.project_name
  bucket_name  = var.shards_bucket_name
}

module "identity" {
  source = "./modules/identity"

  project_name        = var.project_name
  shards_bucket_arn   = module.storage.bucket_arn
  kms_key_arn         = module.storage.kms_key_arn
  dynamodb_table_arns = module.storage.dynamodb_table_arns
  retry_queue_arn     = module.orchestration.queue_arn
}

module "orchestration" {
  source = "./modules/orchestration"

  project_name      = var.project_name
  event_bus_name    = var.event_bus_name
  max_receive_count = var.max_receive_count
}

module "observability" {
  source = "./modules/observability"

  project_name       = var.project_name
  sns_topic_arn      = module.budget.sns_topic_arn
  dlq_arn            = module.orchestration.dlq_arn
  dlq_name           = module.orchestration.dlq_name
  log_retention_days = var.log_retention_days
}
