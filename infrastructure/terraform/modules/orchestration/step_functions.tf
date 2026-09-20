# Placeholder Step Functions state machine mirroring the shard lifecycle:
#   VALIDATE -> FRAGMENTED -> UPLOAD_SHARDS -> VERIFY_CHECKSUMS
#   -> REGISTER_METADATA -> HEALTHY
#
# Every state is a Pass state today because there is no Lambda/task wiring
# yet. In a later phase, real Task states (Lambda ARNs, ECS RunTask, etc.)
# will replace these Pass states without changing the overall lifecycle
# shape.

data "aws_iam_policy_document" "sfn_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "sfn_execution" {
  name               = "${var.project_name}-sfn-execution-role"
  assume_role_policy = data.aws_iam_policy_document.sfn_assume.json

  tags = {
    Project = var.project_name
  }
}

# No task permissions are needed yet since every state is a Pass state.
# When real Task states are added (Lambda invoke, etc.) this role's policy
# must be extended with the corresponding invoke/execute permissions.
data "aws_iam_policy_document" "sfn_execution_policy" {
  statement {
    sid    = "CloudWatchLogsForStateMachine"
    effect = "Allow"
    actions = [
      "logs:CreateLogDelivery",
      "logs:GetLogDelivery",
      "logs:UpdateLogDelivery",
      "logs:DeleteLogDelivery",
      "logs:ListLogDeliveries",
      "logs:PutResourcePolicy",
      "logs:DescribeResourcePolicies",
      "logs:DescribeLogGroups",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "sfn_execution_policy" {
  name   = "${var.project_name}-sfn-execution-policy"
  role   = aws_iam_role.sfn_execution.id
  policy = data.aws_iam_policy_document.sfn_execution_policy.json
}

resource "aws_sfn_state_machine" "shard_lifecycle" {
  name     = "${var.project_name}-shard-lifecycle"
  role_arn = aws_iam_role.sfn_execution.arn

  definition = jsonencode({
    Comment = "ShardX shard lifecycle (placeholder Pass states; real task ARNs are filled in a later phase)"
    StartAt = "VALIDATE"
    States = {
      VALIDATE = {
        Type    = "Pass"
        Comment = "Placeholder for shard validation task"
        Next    = "FRAGMENTED"
      }
      FRAGMENTED = {
        Type    = "Pass"
        Comment = "Placeholder for fragmentation task"
        Next    = "UPLOAD_SHARDS"
      }
      UPLOAD_SHARDS = {
        Type    = "Pass"
        Comment = "Placeholder for shard upload task"
        Next    = "VERIFY_CHECKSUMS"
      }
      VERIFY_CHECKSUMS = {
        Type    = "Pass"
        Comment = "Placeholder for checksum verification task"
        Next    = "REGISTER_METADATA"
      }
      REGISTER_METADATA = {
        Type    = "Pass"
        Comment = "Placeholder for metadata registration task"
        Next    = "HEALTHY"
      }
      HEALTHY = {
        Type = "Succeed"
      }
    }
  })

  tags = {
    Project = var.project_name
  }
}
