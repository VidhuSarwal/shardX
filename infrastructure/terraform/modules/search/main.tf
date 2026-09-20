# search module
#
# Single-node Amazon OpenSearch Service domain, deliberately INSTANCE-BASED
# (t3.small.search) rather than OpenSearch Serverless, so it only bills for
# actual uptime and can be cleanly `terraform destroy`'d between development
# sessions. This module is designed to be applied/destroyed independently
# from the rest of the ShardX infrastructure -- see the root README and the
# top-level search/ directory, which has its own Terraform state.

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

resource "aws_cloudwatch_log_group" "opensearch" {
  name              = "/${var.project_name}/opensearch"
  retention_in_days = 14

  tags = {
    Project = var.project_name
  }
}

# Allow the OpenSearch service to write slow/application logs to the group.
data "aws_iam_policy_document" "opensearch_logs_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["es.amazonaws.com"]
    }

    actions = [
      "logs:PutLogEvents",
      "logs:CreateLogStream",
    ]

    resources = ["${aws_cloudwatch_log_group.opensearch.arn}:*"]
  }
}

resource "aws_cloudwatch_log_resource_policy" "opensearch" {
  policy_name     = "${var.project_name}-opensearch-logs"
  policy_document = data.aws_iam_policy_document.opensearch_logs_policy.json
}

resource "aws_opensearch_domain" "this" {
  domain_name    = var.domain_name
  engine_version = var.engine_version

  cluster_config {
    instance_type  = var.instance_type
    instance_count = 1
    # No dedicated master nodes / no multi-AZ: single-node, cost-minimal
    # for a hackathon-scoped Phase 1. Not for production HA.
    zone_awareness_enabled = false
  }

  ebs_options {
    ebs_enabled = true
    volume_type = "gp3"
    volume_size = var.ebs_volume_size_gb
  }

  encrypt_at_rest {
    enabled = true
  }

  node_to_node_encryption {
    enabled = true
  }

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-2-2019-07"
  }

  # Required for encrypt_at_rest / node_to_node_encryption combined with
  # fine-grained access control being disabled; access is instead
  # restricted via the resource-based access policy below (IAM auth,
  # SigV4-signed requests only -- no anonymous/IP-based access).
  advanced_security_options {
    enabled = false
  }

  log_publishing_options {
    cloudwatch_log_group_arn = aws_cloudwatch_log_group.opensearch.arn
    log_type                 = "ES_APPLICATION_LOGS"
    enabled                  = true
  }

  tags = {
    Project = var.project_name
  }

  depends_on = [aws_cloudwatch_log_resource_policy.opensearch]
}

# Access policy restricted to the api-role / shard-worker-role principals
# passed in as variables. Not open to the public internet.
data "aws_iam_policy_document" "opensearch_access" {
  statement {
    effect = "Allow"

    principals {
      type        = "AWS"
      identifiers = var.allowed_role_arns
    }

    actions   = ["es:ESHttp*"]
    resources = ["${aws_opensearch_domain.this.arn}/*"]
  }
}

resource "aws_opensearch_domain_policy" "this" {
  domain_name     = aws_opensearch_domain.this.domain_name
  access_policies = data.aws_iam_policy_document.opensearch_access.json
}
