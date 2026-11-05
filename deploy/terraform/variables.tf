variable "project_id" {
  type = string
}
variable "region" {
  type    = string
  default = "asia-east1"
}
variable "queue_id" {
  type    = string
  default = "line-bot-events"
}
variable "public_service_name" {
  type    = string
  default = "line-bot-public"
}
variable "worker_service_name" {
  type    = string
  default = "line-bot-private"
}
variable "notification_channels" {
  type    = list(string)
  default = []
}
variable "enable_monitoring" {
  type        = bool
  default     = false
  description = "Creates alert policies only after notification channels and staging thresholds are approved."
}
