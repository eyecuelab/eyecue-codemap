resource "google_artifact_registry_repository" "eyecue-codemap" {
  provider = google-beta
  repository_id = "eyecue-codemap"
  location = "us-central1"
  description = "https://github.com/eyecuelab/codemap"
  format = "DOCKER"
}

resource "google_service_account" "eyecue-codemap" {
  account_id = "eyecue-codemap"
  description = "Used to pull from the eyecue-codemap Artifact Registry repo"
  display_name = "eyecue-codemap"
}

resource "google_service_account" "eyecue-codemap-ci" {
  account_id = "eyecue-codemap-ci"
  description = "Used to push eyecue-codemap to Artifact Registry from CI/CD"
  display_name = "eyecue-codemap-ci"
}

resource "google_artifact_registry_repository_iam_policy" "eyecue-codemap" {
  provider = google-beta
  repository = "projects/eyecue-ops/locations/us-central1/repositories/eyecue-codemap"
  policy_data = data.google_iam_policy.eyecue-codemap.policy_data
}

data "google_iam_policy" "eyecue-codemap" {
  binding {
    role = "roles/artifactregistry.reader"
    members = [
      "serviceAccount:eyecue-codemap@eyecue-ops.iam.gserviceaccount.com",
    ]
  }

  binding {
    role = "roles/artifactregistry.writer"
    members = [
      "serviceAccount:eyecue-codemap-ci@eyecue-ops.iam.gserviceaccount.com",
    ]
  }
}

data "google_iam_workload_identity_pool" "pool" {
  # This pool is defined in the eyecuelab/devops repo
  provider                  = google-beta
  workload_identity_pool_id = "github-actions"
}

resource "google_service_account_iam_binding" "github-actions" {
  service_account_id = google_service_account.eyecue-codemap-ci.id
  role               = "roles/iam.workloadIdentityUser"
  members = [
    "principalSet://iam.googleapis.com/projects/442505215313/locations/global/workloadIdentityPools/github-actions/attribute.repository/eyecuelab/codemap"
  ]
}
