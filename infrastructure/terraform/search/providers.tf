terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # This root is intentionally kept separate from the top-level ShardX
  # root so its state (and the OpenSearch domain it manages) can be
  # applied/destroyed independently -- OpenSearch is the only component
  # in this project with meaningful per-hour cost. See ../README.md.
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
      Component = "search"
    }
  }
}
