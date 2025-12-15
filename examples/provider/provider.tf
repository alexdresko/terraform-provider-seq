terraform {
  required_providers {
    seq = {
      source  = "example/seq"
      version = "0.0.1"
    }
  }
}

provider "seq" {
  server_url = "http://localhost:5342"
  api_key    = "REDACTED"
}

data "seq_health" "this" {}
