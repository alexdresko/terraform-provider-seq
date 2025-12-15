---
page_title: "Seq Provider"
description: |-
  Terraform provider for managing Seq resources using the Seq HTTP API.
---

# Seq Provider

This provider manages Seq resources using the Seq server HTTP API.

## Authentication

The provider authenticates using a Seq API key sent via the `X-Seq-ApiKey` header.

## Example

```terraform
provider "seq" {
  server_url = "http://localhost:5342"
  api_key    = var.seq_api_key
}

data "seq_health" "this" {}

resource "seq_api_key" "ingest" {
  title       = "terraform-ingest"
  permissions = ["Ingest"]
}
```
