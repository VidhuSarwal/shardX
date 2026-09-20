variable "project_name" {
  description = "Project name/prefix used for resource naming."
  type        = string
}

variable "domain_name" {
  description = "Name of the OpenSearch domain."
  type        = string
  default     = "shardx-search"
}

variable "instance_type" {
  description = "Instance type for the single-node OpenSearch domain. Must remain instance-based (not Serverless) to bill only for actual uptime and allow destroy-between-sessions."
  type        = string
  default     = "t3.small.search"
}

variable "ebs_volume_size_gb" {
  description = "EBS volume size (GB) attached to the single OpenSearch node."
  type        = number
  default     = 10
}

variable "engine_version" {
  description = "OpenSearch engine version."
  type        = string
  default     = "OpenSearch_2.13"
}

variable "allowed_role_arns" {
  description = "IAM role ARNs permitted to access the domain (expected: api-role and shard-worker-role ARNs from the identity module's outputs, copied in since this is a separate state root)."
  type        = list(string)
}
