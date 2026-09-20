# storage module
#
# S3 bucket for file shards, encrypted with a dedicated customer-managed
# KMS key. Public access is fully blocked and versioning is enabled.
#
# DynamoDB metadata tables live in dynamodb.tf alongside, sharing the CMK.

data "aws_caller_identity" "current" {}

resource "aws_kms_key" "shards" {
  description             = "CMK for encrypting ${var.project_name} file shard storage"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Project = var.project_name
  }
}

resource "aws_kms_alias" "shards" {
  name          = "alias/${var.project_name}-shards"
  target_key_id = aws_kms_key.shards.key_id
}

resource "aws_s3_bucket" "shards" {
  bucket = var.bucket_name

  tags = {
    Project = var.project_name
  }
}

resource "aws_s3_bucket_versioning" "shards" {
  bucket = aws_s3_bucket.shards.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "shards" {
  bucket = aws_s3_bucket.shards.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.shards.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "shards" {
  bucket = aws_s3_bucket.shards.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
