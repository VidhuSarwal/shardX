output "user_pool_id" {
  description = "Cognito User Pool ID."
  value       = aws_cognito_user_pool.this.id
}

output "user_pool_arn" {
  description = "Cognito User Pool ARN."
  value       = aws_cognito_user_pool.this.arn
}

output "user_pool_client_id" {
  description = "Cognito User Pool Client ID."
  value       = aws_cognito_user_pool_client.this.id
}

output "api_role_arn" {
  description = "ARN of the api-role IAM role."
  value       = aws_iam_role.api_role.arn
}

output "shard_worker_role_arn" {
  description = "ARN of the shard-worker-role IAM role. Consumed by the standalone search/ root module (via manual variable copy, since it has separate state) to authorize OpenSearch access."
  value       = aws_iam_role.shard_worker_role.arn
}

output "security_worker_role_arn" {
  description = "ARN of the security-worker-role IAM role (placeholder for later GuardDuty/Macie phases)."
  value       = aws_iam_role.security_worker_role.arn
}

output "api_policy_json" {
  description = "api-role inline policy document, reused by the compute module's EC2 host role."
  value       = data.aws_iam_policy_document.api_role_policy.json
}

output "shard_worker_policy_json" {
  description = "shard-worker-role inline policy document, reused by the compute module's EC2 host role."
  value       = data.aws_iam_policy_document.shard_worker_role_policy.json
}
