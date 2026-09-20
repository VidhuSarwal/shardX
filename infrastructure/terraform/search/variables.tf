variable "aws_region" {
  description = "AWS region to deploy the OpenSearch domain into. Should match the region used for the main ShardX root module."
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name/prefix used for naming and tagging."
  type        = string
  default     = "shardx"
}

variable "domain_name" {
  description = "Name of the OpenSearch domain."
  type        = string
  default     = "shardx-search"
}

variable "instance_type" {
  description = "Instance type for the single-node OpenSearch domain."
  type        = string
  default     = "t3.small.search"
}

variable "ebs_volume_size_gb" {
  description = "EBS volume size (GB) attached to the single OpenSearch node."
  type        = number
  default     = 10
}

variable "allowed_role_arns" {
  description = "IAM role ARNs permitted to access the domain. Copy the api_role_arn and shard_worker_role_arn values from `terraform output` in the main root module here (this root has separate state and cannot reference module.identity directly)."
  type        = list(string)
}
