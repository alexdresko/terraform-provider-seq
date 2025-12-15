# Terraform Provider for Seq

This repository contains a Terraform provider for managing resources in **Seq** using the Seq HTTP API.

Primary focus: **Seq API Keys** (`/api/apikeys`).

Seq API documentation:
- https://datalust.co/docs/using-the-http-api
- https://datalust.co/docs/server-http-api

## Development

### Requirements

- Go 1.22+
- Terraform 1.5+

### Dev container

This repo includes a devcontainer that starts Seq in Docker alongside the development environment.

- Dev container docs: [.devcontainer/README.md](.devcontainer/README.md)
- Seq UI/API (host): `http://localhost:5342`
- Seq URL from inside the devcontainer: `http://seq:80`

### Build

```powershell
go test ./...
go build -o bin/terraform-provider-seq.exe .
```

### VS Code tasks

Open the Command Palette → **Tasks: Run Task**:
- `go: test`
- `go: fmt`
- `provider: build`
- `docs: generate`

## Provider configuration

```hcl
provider "seq" {
  server_url = "http://localhost:5342"
  api_key    = var.seq_api_key
}
```

Environment variables:
- `SEQ_SERVER_URL`
- `SEQ_API_KEY`
- `SEQ_INSECURE_SKIP_VERIFY`
- `SEQ_TIMEOUT_SECONDS`

## Resources

- `seq_api_key` - manages Seq API keys.

## Data sources

- `seq_health` - reads `/health`.

## Notes

- Seq may only return an API key token on creation. The provider stores the token in state as a **sensitive** attribute and preserves it when Seq does not return it on subsequent reads.
