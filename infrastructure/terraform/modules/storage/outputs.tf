output "bucket_arn" {
  description = "ARN of the S3 bucket used for file shards."
  value       = aws_s3_bucket.shards.arn
}

output "bucket_name" {
  description = "Name of the S3 bucket used for file shards."
  value       = aws_s3_bucket.shards.bucket
}

output "kms_key_arn" {
  description = "ARN of the KMS CMK used to encrypt the shards bucket."
  value       = aws_kms_key.shards.arn
}

output "kms_key_id" {
  description = "Key ID of the KMS CMK used to encrypt the shards bucket."
  value       = aws_kms_key.shards.key_id
}

output "dynamodb_table_arns" {
  description = "ARNs of all DynamoDB metadata tables (for IAM scoping)."
  value = [
    aws_dynamodb_table.users.arn,
    aws_dynamodb_table.oauth_states.arn,
    aws_dynamodb_table.upload_sessions.arn,
    aws_dynamodb_table.drive_accounts.arn,
    aws_dynamodb_table.shard_metadata.arn,
  ]
}

output "dynamodb_table_prefix" {
  description = "Value for DYNAMODB_TABLE_PREFIX in the Go backend."
  value       = local.table_prefix
}
