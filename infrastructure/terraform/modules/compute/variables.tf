variable "project_name" { type = string }
variable "aws_region" { type = string }

variable "instance_type" {
  type    = string
  default = "t3.small"
}

variable "repo_url" {
  description = "Git URL the instance clones (public repo)."
  type        = string
}

variable "repo_ref" {
  description = "Branch/tag/commit the instance checks out."
  type        = string
  default     = "main"
}

variable "app_env" {
  description = "Rendered .env for api + shardworker, minus BASE_URL/FRONTEND_URL which this module sets from the CloudFront domain."
  type        = string
  sensitive   = true
}

variable "api_policy_json" { type = string }
variable "shard_worker_policy_json" { type = string }
variable "user_pool_arn" { type = string }
variable "event_bus_arn" { type = string }
variable "state_machine_arn" { type = string }
variable "log_group_name" { type = string }
