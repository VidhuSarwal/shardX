output "domain_endpoint" {
  description = "Endpoint of the OpenSearch domain."
  value       = module.search.domain_endpoint
}

output "domain_arn" {
  description = "ARN of the OpenSearch domain."
  value       = module.search.domain_arn
}

output "domain_name" {
  description = "Name of the OpenSearch domain."
  value       = module.search.domain_name
}
