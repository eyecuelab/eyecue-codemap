terraform {
  backend "gcs" {
    bucket = "eyecue-ops-terraform"
    credentials = "GCP-serviceacct_eyecue-ops_eyecue-ops-terraform.json"
    prefix = "eyecue-codemap"
  }
}

provider "google-beta" {
  project = "eyecue-ops"
  region  = "us-central1"
  zone    = "us-central1-c"
}
