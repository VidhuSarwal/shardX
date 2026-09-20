resource "aws_sqs_queue" "dlq" {
  name                      = "${var.project_name}-events-dlq"
  message_retention_seconds = 1209600 # 14 days, max retention for triage

  tags = {
    Project = var.project_name
  }
}

resource "aws_sqs_queue" "events" {
  name                       = "${var.project_name}-events"
  visibility_timeout_seconds = 60
  message_retention_seconds  = 345600 # 4 days

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Project = var.project_name
  }
}
