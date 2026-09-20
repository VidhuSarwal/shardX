variable "project_name" {
  description = "Project name/prefix used for resource naming."
  type        = string
}

variable "sns_topic_arn" {
  description = "ARN of the SNS topic (from the budget module) to reuse for alarm notifications."
  type        = string
}

variable "dlq_arn" {
  description = "ARN of the orchestration module's dead-letter queue, used as the dimension for the DLQ-depth alarm."
  type        = string
}

variable "dlq_name" {
  description = "Name of the orchestration module's dead-letter queue, used as the dimension for the DLQ-depth alarm."
  type        = string
}

variable "log_retention_days" {
  description = "Retention period, in days, for the application CloudWatch Log Group."
  type        = number
  default     = 30
}
