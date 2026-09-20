# Basic example alarms, wired to the budget module's SNS topic so we reuse
# a single alerting channel instead of standing up a second one.

# Example 1: DLQ has any visible messages -- indicates the orchestration
# pipeline is failing and messages are being dead-lettered.
resource "aws_cloudwatch_metric_alarm" "dlq_messages_visible" {
  alarm_name          = "${var.project_name}-dlq-messages-visible"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = 300
  statistic           = "Maximum"
  threshold           = 0
  treat_missing_data  = "notBreaching"

  dimensions = {
    QueueName = var.dlq_name
  }

  alarm_description = "Fires when any message lands in the ${var.project_name} dead-letter queue."
  alarm_actions     = [var.sns_topic_arn]
  ok_actions        = [var.sns_topic_arn]

  tags = {
    Project = var.project_name
  }
}

# Example 2: placeholder 5xx-rate style alarm. There is no API Gateway /
# ALB wired up yet in this phase, so this alarm watches a namespace/metric
# placeholder and is expected to be pointed at real dimensions once the API
# layer is provisioned.
resource "aws_cloudwatch_metric_alarm" "api_5xx_rate_placeholder" {
  alarm_name          = "${var.project_name}-api-5xx-rate-placeholder"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "5XXError"
  namespace           = "AWS/ApiGateway"
  period              = 300
  statistic           = "Sum"
  threshold           = 5
  treat_missing_data  = "notBreaching"

  alarm_description = "Placeholder: fires when 5xx count exceeds 5 in a 5-minute window. Attach real ApiName/Stage dimensions once the API layer exists."
  alarm_actions     = [var.sns_topic_arn]
  ok_actions        = [var.sns_topic_arn]

  tags = {
    Project = var.project_name
  }
}
