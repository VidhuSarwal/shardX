# DynamoDB tables backing internal/metadatastore/dynamodb_store.go. The
# schema (keys, GSIs, projections, TTL) is dictated by that file's package
# comment -- keep the two in sync. All tables are on-demand (PAY_PER_REQUEST)
# so idle cost is zero, and use the shards CMK for encryption at rest.

locals {
  table_prefix = var.project_name
  dynamo_tags  = { Project = var.project_name }
}

resource "aws_dynamodb_table" "users" {
  name         = "${local.table_prefix}-users"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
  attribute {
    name = "email"
    type = "S"
  }

  # FindUserByEmail reads password_hash off the GSI: projection must be ALL.
  global_secondary_index {
    name            = "email-index"
    hash_key        = "email"
    projection_type = "ALL"
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.shards.arn
  }
  point_in_time_recovery { enabled = true }
  tags = local.dynamo_tags
}

resource "aws_dynamodb_table" "oauth_states" {
  name         = "${local.table_prefix}-oauth-states"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "state"

  attribute {
    name = "state"
    type = "S"
  }

  # Mongo expires these after 10 min via a TTL index; DynamoDB needs a
  # numeric epoch attribute. The Go store does not populate it yet (see
  # the package comment), so this only takes effect once it does.
  ttl {
    attribute_name = "expires_at_unix"
    enabled        = true
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.shards.arn
  }
  tags = local.dynamo_tags
}

resource "aws_dynamodb_table" "upload_sessions" {
  name         = "${local.table_prefix}-upload-sessions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
  attribute {
    name = "user_id"
    type = "S"
  }

  # CountActiveUserSessions filters on status via the GSI: projection ALL.
  global_secondary_index {
    name            = "user_id-index"
    hash_key        = "user_id"
    projection_type = "ALL"
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.shards.arn
  }
  point_in_time_recovery { enabled = true }
  tags = local.dynamo_tags
}

resource "aws_dynamodb_table" "drive_accounts" {
  name         = "${local.table_prefix}-drive-accounts"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
  attribute {
    name = "user_id"
    type = "S"
  }

  # ListUserDriveAccounts reads encrypted_token off the GSI: projection ALL.
  global_secondary_index {
    name            = "user_id-index"
    hash_key        = "user_id"
    projection_type = "ALL"
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.shards.arn
  }
  point_in_time_recovery { enabled = true }
  tags = local.dynamo_tags
}

resource "aws_dynamodb_table" "shard_metadata" {
  name         = "${local.table_prefix}-shard-metadata"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "file_id"
  range_key    = "shard_id"

  attribute {
    name = "file_id"
    type = "S"
  }
  attribute {
    name = "shard_id"
    type = "N"
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.shards.arn
  }
  point_in_time_recovery { enabled = true }
  tags = local.dynamo_tags
}
