output "event_bus_name" {
  description = "Name of the custom EventBridge event bus."
  value       = aws_cloudwatch_event_bus.this.name
}

output "event_bus_arn" {
  description = "ARN of the custom EventBridge event bus."
  value       = aws_cloudwatch_event_bus.this.arn
}

output "queue_url" {
  description = "URL of the main events SQS queue."
  value       = aws_sqs_queue.events.id
}

output "queue_arn" {
  description = "ARN of the main events SQS queue."
  value       = aws_sqs_queue.events.arn
}

output "dlq_url" {
  description = "URL of the dead-letter SQS queue."
  value       = aws_sqs_queue.dlq.id
}

output "dlq_arn" {
  description = "ARN of the dead-letter SQS queue."
  value       = aws_sqs_queue.dlq.arn
}

output "dlq_name" {
  description = "Name of the dead-letter SQS queue (used as a CloudWatch alarm dimension)."
  value       = aws_sqs_queue.dlq.name
}

output "state_machine_arn" {
  description = "ARN of the placeholder shard-lifecycle Step Functions state machine."
  value       = aws_sfn_state_machine.shard_lifecycle.arn
}
