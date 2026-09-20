resource "aws_cloudwatch_event_bus" "this" {
  name = var.event_bus_name

  tags = {
    Project = var.project_name
  }
}
