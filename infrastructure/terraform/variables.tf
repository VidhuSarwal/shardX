variable "aws_region" {
  description = "AWS region to deploy ShardX infrastructure into."
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name/prefix used for naming and tagging all resources."
  type        = string
  default     = "shardx"
}

variable "budget_alert_email" {
  description = "Email address to receive AWS Budget alert notifications (80% and 100% of monthly threshold)."
  type        = string
}

variable "monthly_budget_usd" {
  description = "Monthly cost threshold, in USD, for the AWS Budget safety net."
  type        = number
  default     = 25
}

variable "shards_bucket_name" {
  description = "Name of the S3 bucket used to store file shards. S3 bucket names are globally unique across ALL AWS accounts, so the default below will almost certainly collide on a real apply -- override it (e.g. with an account ID or random suffix) before applying."
  type        = string
  default     = "shardx-prod-shards"
}

variable "event_bus_name" {
  description = "Name of the custom EventBridge event bus."
  type        = string
  default     = "shardx-events"
}

variable "max_receive_count" {
  description = "Number of times an SQS message can be received before it is sent to the dead-letter queue."
  type        = number
  default     = 5
}

variable "log_retention_days" {
  description = "Retention period, in days, for the application CloudWatch Log Group."
  type        = number
  default     = 30
}
