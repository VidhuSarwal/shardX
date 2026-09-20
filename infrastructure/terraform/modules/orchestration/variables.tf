variable "project_name" {
  description = "Project name/prefix used for resource naming."
  type        = string
}

variable "event_bus_name" {
  description = "Name of the custom EventBridge event bus."
  type        = string
  default     = "shardx-events"
}

variable "max_receive_count" {
  description = "Number of times a message can be received before being sent to the dead-letter queue."
  type        = number
  default     = 5
}
