output "bucket_arn" {
  description = "ARN of the S3 bucket used for file shards."
  value       = aws_s3_bucket.shards.arn
}

output "bucket_name" {
  description = "Name of the S3 bucket used for file shards."
  value       = aws_s3_bucket.shards.bucket
}

output "kms_key_arn" {
  description = "ARN of the KMS CMK used to encrypt the shards bucket."
  value       = aws_kms_key.shards.arn
}

output "kms_key_id" {
  description = "Key ID of the KMS CMK used to encrypt the shards bucket."
  value       = aws_kms_key.shards.key_id
}
