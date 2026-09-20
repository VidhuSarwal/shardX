# Three least-privilege IAM roles, scoped to this project's own resources
# (not account-wide wildcards). All roles that touch S3 objects encrypted
# with the shards KMS CMK also get kms:Decrypt / kms:GenerateDataKey* on
# that key -- without it, S3 reads/writes fail at runtime even though the
# S3 permissions look correct.

data "aws_partition" "current" {}

# ---------------------------------------------------------------------------
# api-role: S3 + DynamoDB read/write scoped to this project's resources
# ---------------------------------------------------------------------------

data "aws_iam_policy_document" "api_role_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com", "ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "api_role" {
  name               = "${var.project_name}-api-role"
  assume_role_policy = data.aws_iam_policy_document.api_role_assume.json

  tags = {
    Project = var.project_name
  }
}

data "aws_iam_policy_document" "api_role_policy" {
  statement {
    sid    = "S3ShardsReadWrite"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket",
    ]
    resources = [
      var.shards_bucket_arn,
      "${var.shards_bucket_arn}/*",
    ]
  }

  statement {
    sid       = "KmsForShardsBucket"
    effect    = "Allow"
    actions   = ["kms:Decrypt", "kms:GenerateDataKey", "kms:GenerateDataKeyWithoutPlaintext", "kms:DescribeKey"]
    resources = [var.kms_key_arn]
  }

  dynamic "statement" {
    for_each = length(var.dynamodb_table_arns) > 0 ? [1] : []
    content {
      sid    = "DynamoDbReadWrite"
      effect = "Allow"
      actions = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan",
        "dynamodb:BatchGetItem",
        "dynamodb:BatchWriteItem",
      ]
      resources = concat(
        var.dynamodb_table_arns,
        [for arn in var.dynamodb_table_arns : "${arn}/index/*"]
      )
    }
  }
}

data "aws_iam_policy_document" "api_role_sqs" {
  count = var.retry_queue_arn != "" ? 1 : 0
  statement {
    sid       = "SqsEnqueueRetries"
    effect    = "Allow"
    actions   = ["sqs:SendMessage", "sqs:GetQueueUrl"]
    resources = [var.retry_queue_arn]
  }
}

resource "aws_iam_role_policy" "api_role_sqs" {
  count  = var.retry_queue_arn != "" ? 1 : 0
  name   = "${var.project_name}-api-role-sqs"
  role   = aws_iam_role.api_role.id
  policy = data.aws_iam_policy_document.api_role_sqs[0].json
}

resource "aws_iam_role_policy" "api_role_policy" {
  name   = "${var.project_name}-api-role-policy"
  role   = aws_iam_role.api_role.id
  policy = data.aws_iam_policy_document.api_role_policy.json
}

# ---------------------------------------------------------------------------
# shard-worker-role: S3 read/write + DynamoDB + SQS consume (cmd/shardworker)
# ---------------------------------------------------------------------------

data "aws_iam_policy_document" "shard_worker_role_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com", "ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "shard_worker_role" {
  name               = "${var.project_name}-shard-worker-role"
  assume_role_policy = data.aws_iam_policy_document.shard_worker_role_assume.json

  tags = {
    Project = var.project_name
  }
}

data "aws_iam_policy_document" "shard_worker_role_policy" {
  statement {
    sid    = "S3ShardsReadWrite"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:ListBucket",
    ]
    resources = [
      var.shards_bucket_arn,
      "${var.shards_bucket_arn}/*",
    ]
  }

  statement {
    sid       = "KmsForShardsBucket"
    effect    = "Allow"
    actions   = ["kms:Decrypt", "kms:GenerateDataKey", "kms:GenerateDataKeyWithoutPlaintext", "kms:DescribeKey"]
    resources = [var.kms_key_arn]
  }

  dynamic "statement" {
    for_each = length(var.dynamodb_table_arns) > 0 ? [1] : []
    content {
      sid       = "DynamoDbShardAndSession"
      effect    = "Allow"
      actions   = ["dynamodb:GetItem", "dynamodb:UpdateItem", "dynamodb:Query"]
      resources = concat(var.dynamodb_table_arns, [for arn in var.dynamodb_table_arns : "${arn}/index/*"])
    }
  }

  dynamic "statement" {
    for_each = var.retry_queue_arn != "" ? [1] : []
    content {
      sid       = "SqsConsumeRetries"
      effect    = "Allow"
      actions   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes", "sqs:GetQueueUrl"]
      resources = [var.retry_queue_arn]
    }
  }
}

resource "aws_iam_role_policy" "shard_worker_role_policy" {
  name   = "${var.project_name}-shard-worker-role-policy"
  role   = aws_iam_role.shard_worker_role.id
  policy = data.aws_iam_policy_document.shard_worker_role_policy.json
}

# ---------------------------------------------------------------------------
# security-worker-role: placeholder minimal policy (CloudWatch Logs only)
#
# Reserved for a later phase that wires up GuardDuty / Macie findings
# processing. For now it only needs to write its own execution logs.
# ---------------------------------------------------------------------------

data "aws_iam_policy_document" "security_worker_role_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "security_worker_role" {
  name               = "${var.project_name}-security-worker-role"
  assume_role_policy = data.aws_iam_policy_document.security_worker_role_assume.json

  tags = {
    Project = var.project_name
  }
}

data "aws_iam_policy_document" "security_worker_role_policy" {
  statement {
    sid    = "CloudWatchLogsWriteOnly"
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["arn:${data.aws_partition.current.partition}:logs:*:*:log-group:/${var.project_name}/*"]
  }
}

resource "aws_iam_role_policy" "security_worker_role_policy" {
  name   = "${var.project_name}-security-worker-role-policy"
  role   = aws_iam_role.security_worker_role.id
  policy = data.aws_iam_policy_document.security_worker_role_policy.json
}
