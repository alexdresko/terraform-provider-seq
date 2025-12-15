# Dev Container

This repo includes a VS Code devcontainer that starts:

- A development container with Go + Terraform + Docker (Docker-in-Docker)
- A Seq container (`datalust/seq:latest`) for local testing

## Quick start

1. In VS Code: **Dev Containers: Reopen in Container**
2. Seq will be available at http://localhost:5342 (forwarded automatically)
3. Sign in with:
   - username: `admin`
   - password: `admin`

## Provider configuration (inside the devcontainer)

The devcontainer sets `SEQ_SERVER_URL=http://seq:80` so Terraform running inside the container can reach Seq via the Docker network.

Example:

```hcl
provider "seq" {
  server_url = "http://seq:80"
  api_key    = var.seq_api_key
}
```

## Creating an API key

In the Seq UI:

1. Settings → API Keys
2. Create a key (copy the token)
3. Use the token as `api_key` when configuring the Terraform provider

## Notes

- `SEQ_PASSWORD` is set to a simple dev-only value in `.devcontainer/docker-compose.yml`.
- Data is persisted in a Docker volume `seq-data`.
- Docker daemon data for the dev container is persisted in `docker-data`.
