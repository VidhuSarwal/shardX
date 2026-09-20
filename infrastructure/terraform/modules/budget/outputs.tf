output "sns_topic_arn" {
  description = "ARN of the SNS topic used for budget alerts. Reused by the observability module for alarm notifications."
  value       = aws_sns_topic.budget_alerts.arn
}

output "budget_name" {
  description = "Name of the AWS Budget resource."
  value       = aws_budgets_budget.monthly.name
}
