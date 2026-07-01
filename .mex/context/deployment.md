---
name: deployment
description: AWS infrastructure, CI/CD, and release operations (Terraform + GitHub Actions + NGINX/Certbot on EC2). Load when deploying, changing infra, or debugging staging/prod.
triggers:
  - "deploy"
  - "deployment"
  - "terraform"
  - "infra"
  - "ci/cd"
  - "github actions"
  - "staging"
  - "production"
  - "nginx"
  - "ec2"
  - "release"
edges:
  - target: context/setup.md
    condition: when mapping local env vars to what the deploy writes on the instance
  - target: context/decisions.md
    condition: when the change relates to the generated-CSS / build-at-deploy decision
  - target: context/architecture.md
    condition: when you need how the running server is structured behind NGINX
last_updated: 2026-06-26
---

# Deployment & Infrastructure

Two environments — **staging** and **production** — on AWS (`ap-south-1`), provisioned by Terraform and
shipped by GitHub Actions. Authoritative detail lives in `terraform/` and `.github/workflows/`.

## Topology

- A single **EC2** instance per env (Amazon Linux 2023, ARM64): staging `t4g.small`, prod `t4g.medium`.
- **NGINX** on the instance terminates TLS and reverse-proxies to the Go server on `127.0.0.1:8080`.
- The Go app runs as a **systemd** service `nexgencms` under the `app` user, `WorkingDirectory=/opt/nex-gen-cms`.
- An **Elastic IP** is attached; **Cloudflare** holds DNS (single A record, `proxied=false` so Let's Encrypt
  HTTP-01 works). TLS via **Certbot** (auto-renew with an NGINX reload hook).
- Domains: staging `staging-<subdomain>.<zone>` (e.g. `staging-new-cms.avantifellows.org`); prod
  `<subdomain>.<zone>` (`new-cms.avantifellows.org`).

## Terraform (`terraform/`)

- `main.tf` — EC2, security group (22 from `ssh_cidr`; 80/443 from anywhere), EIP, Cloudflare A record.
- `variables.tf` — `environment`, `instance_type`, Cloudflare creds, `letsencrypt_email`, `repo_url`/`repo_branch`,
  `db_service_endpoint`/`db_service_token` (sensitive), etc.
- `backend.tf` — remote state in S3 bucket `tfstate-nex-gen-cms` (keys `nex-gen-cms/staging.tfstate`,
  `nex-gen-cms/prod.tfstate`) + DynamoDB lock table `tfstate-nex-gen-cms-locks`.
- `user-data.sh` — **idempotent, runs on every boot**: installs packages (incl. Node for the Tailwind build,
  `fontconfig` for PDF fonts), clones/`hard-reset`s the repo to `repo_branch`, writes `.env`, builds the
  app (`go build ./cmd`) and CSS, (re)creates the systemd unit + NGINX config, and runs Certbot.
- `bootstrap.sh` — one-time creation of the S3 bucket + DynamoDB table. `outputs.tf` — IPs, `application_url`, etc.

## CI/CD (`.github/workflows/`)

- `deploy-staging.yml` — deploys on **push to `main`** (and `workflow_dispatch`).
- `deploy-prod.yml` — deploys on **push to `release`** (backend key `nex-gen-cms/prod.tfstate`).
- Each job: `terraform init/validate/plan/apply` with `TF_VAR_*` from GitHub Secrets. **If Terraform detects
  no infra drift, it reboots the EC2 instance** — `user-data.sh` then pulls the latest code and restarts the
  app. So a code-only release = merge to the branch → reboot.
- Required secrets: `AWS_ACCESS_KEY_ID`/`SECRET`, Cloudflare email/API key/zone, Let's Encrypt email,
  `TF_VAR_db_service_endpoint`/`token` (per env). App config (`DB_SERVICE_*` and the auth vars) is supplied
  via `TF_VAR_*`, never committed.

## Common Operations

- **Ship a code change:** merge to `main` (staging) or `release` (prod); the workflow reboots the box to pull it.
- **First-time setup:** `cd terraform && ./bootstrap.sh`, uncomment the S3 backend, `terraform init -migrate-state`,
  set vars, `terraform apply` (or push to the branch).
- **Access the box:** the `ssh_command` Terraform output if a key pair is set, else AWS SSM Session Manager.
- **Logs:** `/var/log/nexgencms-setup.log` (boot), `journalctl -u nexgencms -f` (app),
  `/var/log/nginx/{error,access}.log`, `sudo certbot certificates` (TLS).

## Gotchas

- **The deployed `.env` must include the auth vars** (`DATABASE_URL`, `GOOGLE_*`, `OAUTH_REDIRECT_URL`,
  `SESSION_SECRET`, `APP_ENV=production`) — not just `DB_SERVICE_*`. The old `CMS_USERNAME`/`PASSWORD` were
  dropped (2026-06-01); prod mirrors staging's env var set. Set `APP_ENV=production` so cookies are `Secure`.
- **CSS is built on the box** (`user-data.sh`), which is why the instance needs **Node 22**; an older Node
  breaks the Tailwind v4 build and the app ships unstyled. See `context/decisions.md`.
- **No zero-downtime deploys** — updates are in-place via reboot; single instance per env, no ASG/HA.
- **Cloudflare must stay `proxied=false`** during ACME issuance/renewal or the HTTP-01 challenge fails.
- **PDFs need Chrome + fonts on the box** — Playwright Chromium under `/opt/playwright-browsers` and
  `fontconfig`; missing either breaks `/download-pdf`. See `patterns/generate-pdf.md`.
