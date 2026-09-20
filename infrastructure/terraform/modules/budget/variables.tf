variable "project_name" {
  description = "Project name/prefix used for resource naming."
  type        = string
}

variable "monthly_budget_usd" {
  description = "Monthly cost threshold in USD for the AWS Budget."
  type        = number
  default     = 25
}

variable "alert_email" {
  description = "Email address to receive budget alert notifications via SNS."
  type        = string
}
