terraform {
  backend "gcs" {
    bucket = "eyecue-ops-terraform"
    prefix = "eyecue-codemap"
  }
}

provider "google" {
  project = "eyecue-ops"
  region  = "us-central1"
  zone    = "us-central1-c"
}

provider "google-beta" {
  project = "eyecue-ops"
  region  = "us-central1"
  zone    = "us-central1-c"
}
