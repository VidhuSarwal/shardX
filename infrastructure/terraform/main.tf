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

# ---------------------------------------------------------------------------
# 6. compute -- EC2 host (api + shardworker + MongoDB) and CloudFront (web +
#    API on one origin). Consumes everything above.
# ---------------------------------------------------------------------------

resource "random_password" "jwt_secret" {
  length  = 48
  special = false
}

resource "random_bytes" "token_enc_key" {
  length = 32
}

module "compute" {
  source = "./modules/compute"

  project_name  = var.project_name
  aws_region    = var.aws_region
  instance_type = var.instance_type
  repo_url      = var.repo_url
  repo_ref      = var.repo_ref

  api_policy_json          = module.identity.api_policy_json
  shard_worker_policy_json = module.identity.shard_worker_policy_json
  user_pool_arn            = module.identity.user_pool_arn
  event_bus_arn            = module.orchestration.event_bus_arn
  state_machine_arn        = module.orchestration.state_machine_arn
  log_group_name           = module.observability.log_group_name

  app_env = <<-EOT
    AWS_REGION=${var.aws_region}
    STORAGE_PROVIDER=s3
    S3_BUCKET=${module.storage.bucket_name}
    S3_KMS_KEY_ID=${module.storage.kms_key_arn}
    DB_PROVIDER=dynamodb
    DYNAMODB_TABLE_PREFIX=${module.storage.dynamodb_table_prefix}
    AUTH_PROVIDER=cognito
    COGNITO_USER_POOL_ID=${module.identity.user_pool_id}
    COGNITO_CLIENT_ID=${module.identity.user_pool_client_id}
    EVENT_BUS_NAME=${module.orchestration.event_bus_name}
    STATE_MACHINE_ARN=${module.orchestration.state_machine_arn}
    SQS_QUEUE_URL=${module.orchestration.queue_url}
    JWT_SECRET=${random_password.jwt_secret.result}
    TOKEN_ENC_KEY=${random_bytes.token_enc_key.base64}
    GOOGLE_CLIENT_ID=${var.google_client_id}
    GOOGLE_CLIENT_SECRET=${var.google_client_secret}
    UPLOAD_TEMP_DIR=/data/uploads
    MAX_FILE_SIZE_GB=${var.max_file_size_gb}
    SESSION_EXPIRY_HOURS=24
    MAX_CONCURRENT_UPLOADS_PER_USER=3
    TEMP_FILE_CLEANUP_MINUTES=60
  EOT
}
