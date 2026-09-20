output "app_url" {
  value = "https://${aws_cloudfront_distribution.this.domain_name}"
}
output "distribution_id" {
  value = aws_cloudfront_distribution.this.id
}
output "web_bucket_name" {
  value = aws_s3_bucket.web.bucket
}
output "instance_id" {
  value = aws_instance.host.id
}
output "app_env_parameter" {
  value = aws_ssm_parameter.app_env.name
}
