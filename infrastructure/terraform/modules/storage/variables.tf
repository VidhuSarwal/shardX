variable "project_name" {
  description = "Project name/prefix used for resource naming."
  type        = string
}

variable "bucket_name" {
  description = "Name of the S3 bucket used to store file shards."
  type        = string
  default     = "shardx-prod-shards"
}
