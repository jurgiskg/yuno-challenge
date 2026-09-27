# Deployment

The backend deploys to [Render](https://render.com) as a Docker web service.
Every push to `main` that touches `backend/**` or `infra/**` triggers a deploy.

| File | Purpose |
|---|---|
| `render.yaml` | Render Blueprint: service, plan, env vars, health check |
| `Dockerfile` | Multi-stage build → static binary on `distroless/static:nonroot` |
| `Dockerfile.dockerignore` | Keeps local binaries and `.env` files out of the build context |

## One-time setup

1. Render dashboard → **New → Blueprint**.
2. Connect GitHub. When installing the Render GitHub App, choose
   **Only select repositories → `yuno-challenge`**.
3. Select the repo, branch `main`, and set **Blueprint Path** to `infra/render.yaml`.
4. Apply. Render generates `API_KEY` (a random 256-bit value) on the first sync.

## Security

- `/v1/health` is public so Render's health checks work. Every other `/v1/*` route
  requires an `X-API-Key` header.
- The service refuses to start with `GIN_MODE=release` (the default in the image)
  if `API_KEY` is empty, so a missing secret fails closed.
- To read or rotate the key, go to the service's **Environment** tab. Rotating it
  restarts the service.
- Don't send the key through the repo or README. Share it with reviewers directly.

## Free plan caveats

- Sleeps after 15 minutes without traffic. The first request after that takes about a minute.
- Storage is in-memory, so sleeping, restarting, or redeploying wipes all attempt logs.

## Run the image locally

```sh
docker build -f infra/Dockerfile -t yuno-failover-api backend
docker run --rm -p 8080:8080 -e API_KEY=dev-key yuno-failover-api

curl localhost:8080/v1/health
curl -H 'X-API-Key: dev-key' -X POST localhost:8080/v1/authorizations -d @request.json
```

To run without Docker, copy `backend/.env.example` to `backend/.env` (gitignored)
and run `make run` from `backend/`. It loads `PORT`, `API_KEY`, `ACQUIRER_ORDER` and
`GIN_MODE` from that file. An empty `API_KEY` disables auth, for local development only.
