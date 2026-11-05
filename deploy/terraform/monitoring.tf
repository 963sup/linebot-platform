resource "google_monitoring_dashboard" "delivery" {
  count          = var.enable_monitoring ? 1 : 0
  dashboard_json = jsonencode({
    displayName = "LINE durable delivery"
    mosaicLayout = {
      columns = 12
      tiles   = []
    }
  })
}

resource "google_monitoring_alert_policy" "worker_5xx" {
  count        = var.enable_monitoring ? 1 : 0
  display_name = "LINE worker 5xx"
  combiner     = "OR"
  notification_channels = var.notification_channels
  conditions {
    display_name = "Private worker 5xx responses"
    condition_threshold {
      filter          = "resource.type=\"cloud_run_revision\" AND resource.label.service_name=\"${var.worker_service_name}\" AND metric.type=\"run.googleapis.com/request_count\" AND metric.label.response_code_class=\"5xx\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "60s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }
    }
  }
}

resource "google_monitoring_alert_policy" "queue_depth" {
  count        = var.enable_monitoring ? 1 : 0
  display_name = "LINE queue depth"
  combiner     = "OR"
  notification_channels = var.notification_channels
  conditions {
    display_name = "Queue depth remains nonzero"
    condition_threshold {
      filter          = "resource.type=\"cloud_tasks_queue\" AND resource.label.queue_id=\"${var.queue_id}\" AND metric.type=\"cloudtasks.googleapis.com/queue/depth\""
      comparison      = "COMPARISON_GT"
      threshold_value = 100
      duration        = "300s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }
}
