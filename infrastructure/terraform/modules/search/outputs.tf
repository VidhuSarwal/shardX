output "domain_endpoint" {
  description = "Endpoint of the OpenSearch domain."
  value       = aws_opensearch_domain.this.endpoint
}

output "domain_arn" {
  description = "ARN of the OpenSearch domain."
  value       = aws_opensearch_domain.this.arn
}

output "domain_name" {
  description = "Name of the OpenSearch domain."
  value       = aws_opensearch_domain.this.domain_name
}
