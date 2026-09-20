output "shards_bucket_name" {
  description = "Name of the S3 bucket used for file shards."
  value       = module.storage.bucket_name
}

output "shards_bucket_arn" {
  description = "ARN of the S3 bucket used for file shards."
  value       = module.storage.bucket_arn
}

output "kms_key_arn" {
  description = "ARN of the KMS CMK used to encrypt the shards bucket."
  value       = module.storage.kms_key_arn
}

output "cognito_user_pool_id" {
  description = "Cognito User Pool ID."
  value       = module.identity.user_pool_id
}

output "cognito_user_pool_client_id" {
  description = "Cognito User Pool Client ID."
  value       = module.identity.user_pool_client_id
}

output "api_role_arn" {
  description = "ARN of the api-role IAM role."
  value       = module.identity.api_role_arn
}

output "shard_worker_role_arn" {
  description = "ARN of the shard-worker-role IAM role. Copy this into the standalone search/ module's terraform.tfvars (as part of allowed_role_arns) since it has separate state."
  value       = module.identity.shard_worker_role_arn
}

output "security_worker_role_arn" {
  description = "ARN of the security-worker-role IAM role (placeholder for later GuardDuty/Macie phases)."
  value       = module.identity.security_worker_role_arn
}

output "event_bus_name" {
  description = "Name of the custom EventBridge event bus."
  value       = module.orchestration.event_bus_name
}

output "queue_url" {
  description = "URL of the main events SQS queue."
  value       = module.orchestration.queue_url
}

output "dlq_url" {
  description = "URL of the dead-letter SQS queue."
  value       = module.orchestration.dlq_url
}

output "state_machine_arn" {
  description = "ARN of the placeholder shard-lifecycle Step Functions state machine."
  value       = module.orchestration.state_machine_arn
}

output "cloudtrail_arn" {
  description = "ARN of the CloudTrail trail."
  value       = module.observability.trail_arn
}

output "app_log_group_name" {
  description = "Name of the application CloudWatch Log Group."
  value       = module.observability.log_group_name
}

output "dynamodb_table_prefix" {
  description = "Value for DYNAMODB_TABLE_PREFIX in the Go backend (.env)."
  value       = module.storage.dynamodb_table_prefix
}
