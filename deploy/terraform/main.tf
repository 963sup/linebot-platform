locals {
  service_accounts = {
    deployer = "line-bot-deployer"
    public   = "line-bot-public"
    worker   = "line-bot-worker"
    miniapp  = "line-miniapp"
    invoker  = "line-bot-invoker"
  }
}

resource "google_service_account" "runtime" {
  for_each     = local.service_accounts
  account_id   = each.value
  display_name = "Line Bot ${each.key}"
}

resource "google_cloud_tasks_queue" "events" {
  name     = var.queue_id
  location = var.region
  rate_limits {
    max_concurrent_dispatches = 16
    max_dispatches_per_second = 16
  }
  retry_config {
    max_attempts       = 20
    min_backoff        = "1s"
    max_backoff        = "300s"
    max_doublings      = 5
  }
}

# Project roles are intentionally narrow. Secret access is granted per-secret
# below so the public identity cannot read the LINE send credential.
resource "google_project_iam_member" "public_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.runtime["public"].email}"
}
resource "google_project_iam_member" "public_tasks" {
  project = var.project_id
  role    = "roles/cloudtasks.enqueuer"
  member  = "serviceAccount:${google_service_account.runtime["public"].email}"
}
resource "google_project_iam_member" "worker_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.runtime["worker"].email}"
}
resource "google_project_iam_member" "worker_tasks" {
  project = var.project_id
  role    = "roles/cloudtasks.enqueuer"
  member  = "serviceAccount:${google_service_account.runtime["worker"].email}"
}
resource "google_project_iam_member" "miniapp_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.runtime["miniapp"].email}"
}

resource "google_project_iam_member" "deployer_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.runtime["deployer"].email}"
}
resource "google_project_iam_member" "deployer_artifact_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.runtime["deployer"].email}"
}
resource "google_project_iam_member" "deployer_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.runtime["deployer"].email}"
}
resource "google_service_account_iam_member" "deployer_uses_runtime" {
  for_each           = toset(["public", "worker", "miniapp"])
  service_account_id = google_service_account.runtime[each.value].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.runtime["deployer"].email}"
}

data "google_secret_manager_secret" "channel_secret" { secret_id = "line-bot-channel-secret" }
data "google_secret_manager_secret" "access_token" { secret_id = "line-bot-channel-access-token" }
data "google_secret_manager_secret" "attendance_location_key" { secret_id = "attendance-location-key" }

resource "google_secret_manager_secret_iam_member" "public_channel_secret" {
  secret_id = data.google_secret_manager_secret.channel_secret.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime["public"].email}"
}
resource "google_secret_manager_secret_iam_member" "worker_access_token" {
  secret_id = data.google_secret_manager_secret.access_token.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime["worker"].email}"
}
resource "google_secret_manager_secret_iam_member" "miniapp_attendance_location_key" {
  secret_id = data.google_secret_manager_secret.attendance_location_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime["miniapp"].email}"
}
resource "google_secret_manager_secret_iam_member" "miniapp_attendance_location_key_metadata" {
  secret_id = data.google_secret_manager_secret.attendance_location_key.id
  role      = "roles/secretmanager.viewer"
  member    = "serviceAccount:${google_service_account.runtime["miniapp"].email}"
}

# Cloud Run services are deployed from immutable images by Cloud Build. These
# IAM resources attach only after those service names exist.
resource "google_cloud_run_v2_service_iam_member" "task_invoker" {
  project  = var.project_id
  location = var.region
  name     = var.worker_service_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.runtime["invoker"].email}"
}

resource "google_service_account_iam_member" "tasks_mint_oidc" {
  service_account_id = google_service_account.runtime["invoker"].name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-cloudtasks.iam.gserviceaccount.com"
}

resource "google_service_account_iam_member" "enqueue_uses_invoker" {
  for_each           = toset(["public", "worker"])
  service_account_id = google_service_account.runtime["invoker"].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.runtime[each.value].email}"
}

data "google_project" "current" { project_id = var.project_id }
