variable "project_name" {
  description = "Project name/prefix used for resource naming."
  type        = string
}

variable "shards_bucket_arn" {
  description = "ARN of the S3 shards bucket (from the storage module). Passed as an input rather than a module reference to avoid a circular dependency."
  type        = string
}

variable "kms_key_arn" {
  description = "ARN of the KMS CMK used to encrypt the shards bucket (from the storage module). Roles that read/write S3 objects need kms:Decrypt/GenerateDataKey* on this key or S3 calls will 403 at runtime."
  type        = string
}

variable "dynamodb_table_arns" {
  description = "ARNs of DynamoDB tables the api-role and shard-worker-role may read/write."
  type        = list(string)
  default     = []
}

variable "retry_queue_arn" {
  description = "ARN of the SQS shard-retry queue (from the orchestration module). api-role sends to it; shard-worker-role consumes it. Empty disables the SQS statements."
  type        = string
  default     = ""
}
