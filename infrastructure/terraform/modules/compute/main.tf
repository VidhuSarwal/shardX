# Single EC2 host (api + shardworker via docker compose) behind one
# CloudFront distribution that also serves the web app from a private S3
# bucket. Same origin for both, so no CORS and BASE_URL == FRONTEND_URL.

data "aws_partition" "current" {}
data "aws_caller_identity" "current" {}

locals {
  # Referenced by name (not resource) from user_data to avoid a cycle:
  # instance -> EIP -> CloudFront -> parameter -> instance.
  param_name = "/${var.project_name}/app-env"
  param_arn  = "arn:${data.aws_partition.current.partition}:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter${local.param_name}"
}

# ---------------------------------------------------------------------------
# Instance role: api + worker policies from the identity module, plus what
# the API needs that those roles never had (Cognito, EventBridge, Step
# Functions), plus SSM for Session Manager + reading the env parameter.
# ---------------------------------------------------------------------------

data "aws_iam_policy_document" "ec2_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "host" {
  name               = "${var.project_name}-host-role"
  assume_role_policy = data.aws_iam_policy_document.ec2_assume.json
}

resource "aws_iam_role_policy" "api" {
  name   = "api"
  role   = aws_iam_role.host.id
  policy = var.api_policy_json
}

resource "aws_iam_role_policy" "worker" {
  name   = "shard-worker"
  role   = aws_iam_role.host.id
  policy = var.shard_worker_policy_json
}

data "aws_iam_policy_document" "host_extra" {
  statement {
    sid       = "Cognito"
    actions   = ["cognito-idp:SignUp", "cognito-idp:AdminConfirmSignUp", "cognito-idp:InitiateAuth", "cognito-idp:GetUser"]
    resources = [var.user_pool_arn]
  }
  statement {
    sid       = "EventBridge"
    actions   = ["events:PutEvents"]
    resources = [var.event_bus_arn]
  }
  statement {
    sid       = "StepFunctions"
    actions   = ["states:StartExecution"]
    resources = [var.state_machine_arn]
  }
  statement {
    sid       = "EnvParameter"
    actions   = ["ssm:GetParameter"]
    resources = [local.param_arn]
  }
  statement {
    sid       = "Logs"
    actions   = ["logs:CreateLogStream", "logs:PutLogEvents", "logs:DescribeLogStreams"]
    resources = ["arn:${data.aws_partition.current.partition}:logs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:log-group:${var.log_group_name}:*"]
  }
}

resource "aws_iam_role_policy" "host_extra" {
  name   = "host-extra"
  role   = aws_iam_role.host.id
  policy = data.aws_iam_policy_document.host_extra.json
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.host.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "host" {
  name = "${var.project_name}-host"
  role = aws_iam_role.host.name
}

# ---------------------------------------------------------------------------
# App env lives in SSM Parameter Store; the instance polls for it at boot
# (it needs the CloudFront domain, which only exists after the distribution).
# ---------------------------------------------------------------------------

resource "aws_ssm_parameter" "app_env" {
  name  = local.param_name
  type  = "SecureString"
  value = <<-EOT
    ${chomp(var.app_env)}
    BASE_URL=https://${aws_cloudfront_distribution.this.domain_name}
    FRONTEND_URL=https://${aws_cloudfront_distribution.this.domain_name}
  EOT
}

# ---------------------------------------------------------------------------
# EC2
# ---------------------------------------------------------------------------

data "aws_vpc" "default" {
  default = true
}

data "aws_ssm_parameter" "al2023_ami" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
}

data "aws_ec2_managed_prefix_list" "cloudfront" {
  name = "com.amazonaws.global.cloudfront.origin-facing"
}

resource "aws_security_group" "host" {
  name        = "${var.project_name}-host"
  description = "HTTP from CloudFront only; no SSH (use SSM Session Manager)"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    prefix_list_ids = [data.aws_ec2_managed_prefix_list.cloudfront.id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "host" {
  ami                    = data.aws_ssm_parameter.al2023_ami.value
  instance_type          = var.instance_type
  iam_instance_profile   = aws_iam_instance_profile.host.name
  vpc_security_group_ids = [aws_security_group.host.id]
  user_data = templatefile("${path.module}/user_data.sh", {
    repo_url   = var.repo_url
    repo_ref   = var.repo_ref
    param_name = local.param_name
    region     = var.aws_region
  })
  user_data_replace_on_change = true

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
    encrypted   = true
  }
  metadata_options {
    http_tokens = "required"
  }
  tags = { Name = "${var.project_name}-host" }
}

resource "aws_eip" "host" {
  instance = aws_instance.host.id
  tags     = { Name = "${var.project_name}-host" }
}

# ---------------------------------------------------------------------------
# Web bucket (private, OAC) + CloudFront with two origins
# ---------------------------------------------------------------------------

resource "aws_s3_bucket" "web" {
  bucket        = "${var.project_name}-${data.aws_caller_identity.current.account_id}-web"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "web" {
  bucket                  = aws_s3_bucket.web.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_cloudfront_origin_access_control" "web" {
  name                              = "${var.project_name}-web"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

data "aws_iam_policy_document" "web_bucket" {
  statement {
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.web.arn}/*"]
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.this.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "web" {
  bucket = aws_s3_bucket.web.id
  policy = data.aws_iam_policy_document.web_bucket.json
}

# AWS-managed policies
data "aws_cloudfront_cache_policy" "optimized" { name = "Managed-CachingOptimized" }
data "aws_cloudfront_cache_policy" "disabled" { name = "Managed-CachingDisabled" }
data "aws_cloudfront_origin_request_policy" "all_viewer" { name = "Managed-AllViewerExceptHostHeader" }

locals {
  api_paths = ["/api/*", "/oauth2/*", "/health"]
}

resource "aws_cloudfront_distribution" "this" {
  enabled             = true
  comment             = "${var.project_name}: web (S3) + API (EC2)"
  default_root_object = "index.html"
  price_class         = "PriceClass_100"
  http_version        = "http2and3"

  origin {
    origin_id                = "web"
    domain_name              = aws_s3_bucket.web.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.web.id
  }

  origin {
    origin_id   = "api"
    domain_name = aws_eip.host.public_dns
    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1.2"]
      origin_read_timeout    = 60
    }
  }

  default_cache_behavior {
    target_origin_id       = "web"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = data.aws_cloudfront_cache_policy.optimized.id
    compress               = true
  }

  dynamic "ordered_cache_behavior" {
    for_each = local.api_paths
    content {
      path_pattern             = ordered_cache_behavior.value
      target_origin_id         = "api"
      viewer_protocol_policy   = "redirect-to-https"
      allowed_methods          = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
      cached_methods           = ["GET", "HEAD"]
      cache_policy_id          = data.aws_cloudfront_cache_policy.disabled.id
      origin_request_policy_id = data.aws_cloudfront_origin_request_policy.all_viewer.id
    }
  }

  # SPA: unknown paths (e.g. /files/<id>, /oauth/finished) fall through to index.html
  custom_error_response {
    error_code         = 403
    response_code      = 200
    response_page_path = "/index.html"
  }
  custom_error_response {
    error_code         = 404
    response_code      = 200
    response_page_path = "/index.html"
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }
  viewer_certificate {
    cloudfront_default_certificate = true
  }
}
