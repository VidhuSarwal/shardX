output "log_group_name" {
  description = "Name of the application CloudWatch Log Group."
  value       = aws_cloudwatch_log_group.app.name
}

output "trail_arn" {
  description = "ARN of the CloudTrail trail."
  value       = aws_cloudtrail.this.arn
}

output "trail_bucket_name" {
  description = "Name of the S3 bucket used for CloudTrail log delivery."
  value       = aws_s3_bucket.trail_logs.bucket
}
