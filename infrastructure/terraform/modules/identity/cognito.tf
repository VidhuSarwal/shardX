# Cognito User Pool providing JWT auth via InitiateAuth (USER_PASSWORD_AUTH).
# No domain / hosted UI is configured -- this phase only needs token issuance
# for API callers, not a hosted sign-in page.

resource "aws_cognito_user_pool" "this" {
  name = "${var.project_name}-user-pool"

  password_policy {
    minimum_length    = 12
    require_lowercase = true
    require_uppercase = true
    require_numbers   = true
    require_symbols   = true
  }

  auto_verified_attributes = ["email"]

  username_attributes = ["email"]

  admin_create_user_config {
    allow_admin_create_user_only = false
  }

  tags = {
    Project = var.project_name
  }
}

resource "aws_cognito_user_pool_client" "this" {
  name         = "${var.project_name}-client"
  user_pool_id = aws_cognito_user_pool.this.id

  explicit_auth_flows = [
    "ALLOW_USER_PASSWORD_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
    "ALLOW_USER_SRP_AUTH",
  ]

  generate_secret               = false
  prevent_user_existence_errors = "ENABLED"

  access_token_validity  = 1
  id_token_validity      = 1
  refresh_token_validity = 30

  token_validity_units {
    access_token  = "hours"
    id_token      = "hours"
    refresh_token = "days"
  }
}
