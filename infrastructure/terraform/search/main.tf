module "search" {
  source = "../modules/search"

  project_name       = var.project_name
  domain_name        = var.domain_name
  instance_type      = var.instance_type
  ebs_volume_size_gb = var.ebs_volume_size_gb
  allowed_role_arns  = var.allowed_role_arns
}
