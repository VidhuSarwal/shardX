terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Credentials are NOT configured here. Set the AWS_PROFILE environment
# variable before running any terraform command, e.g.:
#
#   export AWS_PROFILE=shardx
#
# Do not hardcode a profile name or credentials in this file.
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project   = var.project_name
      ManagedBy = "terraform"
    }
  }
}
