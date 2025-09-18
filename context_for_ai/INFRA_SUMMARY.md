# Nex Gen CMS — Staging & Production Infrastructure Summary

This document explains the staging and production infrastructure and CI/CD setup provisioned via Terraform and GitHub Actions for the Nex Gen CMS. It complements the application overview in `context_for_ai/PROJECT_OVERVIEW.md` by focusing exclusively on infrastructure. Hardening notes are included where relevant.

## Scope

- Environments: staging, production
- Cloud: AWS (ap-south-1)
- DNS: Cloudflare
- SSL/TLS: Let's Encrypt via Certbot
- CI/CD: GitHub Actions (`.github/workflows/deploy-staging.yml`, `.github/workflows/deploy-prod.yml`)
- IaC: Terraform (`terraform/` directory)

## High-level Architecture

- An EC2 instance (Amazon Linux 2023, ARM64) runs the Go server on port 8080.
- NGINX on the instance terminates TLS and reverse proxies HTTPS/HTTP requests to the app.
- An Elastic IP (EIP) is associated with the instance.
- Cloudflare hosts DNS; a single A record points the staging domain to the EIP.
- TLS certificates are provisioned and renewed by Certbot (Let's Encrypt) on the instance.
- Terraform state is stored remotely in S3 with DynamoDB for state locking.
- GitHub Actions plans/applies Terraform and, when there are no infra changes, reboots the instance to pull the latest code and restart the app.

Environment specifics:

- Staging: instance type `t4g.small`, domain pattern `staging-<subdomain>.<zone>` (e.g., `staging-new-cms.avantifellows.org`).
- Production: instance type `t4g.medium`, domain `<subdomain>.<zone>` (currently `new-cms.avantifellows.org`).

## Terraform Components (terraform/)

### Providers and Backend — `backend.tf`

- Providers: `aws ~> 5.0`, `cloudflare ~> 4.0`.
- Remote state backend (S3 + DynamoDB):
  - Bucket: `tfstate-nex-gen-cms`
  - Keys: `nex-gen-cms/staging.tfstate`, `nex-gen-cms/prod.tfstate`
  - Table: `tfstate-nex-gen-cms-locks`
  - Region: `ap-south-1`
- Workflow:
  1) Run `terraform/bootstrap.sh` once to create the S3 bucket and DynamoDB table.
  2) Uncomment the `backend "s3"` block in `backend.tf` and run `terraform init -migrate-state`.

### Variables — `variables.tf`

Key inputs (all configurable via `terraform.tfvars` or TF_VAR env vars):

- AWS/Env: `aws_region` (default `ap-south-1`), `environment` (`staging|prod`), `ssh_cidr` (default `0.0.0.0/0` for staging only).
- Cloudflare: `cloudflare_email`, `cloudflare_api_key` (sensitive), `cloudflare_zone_name`, `subdomain`.
- TLS: `letsencrypt_email`.
- App/Repo: `repo_url`, `repo_branch`.
- App config: `db_service_endpoint` (sensitive), `db_service_token` (sensitive).
- EC2: `instance_type` (default `t4g.small`), `key_pair_name` (optional; `null` to rely on SSM session manager or disable SSH logins by key pair).

### Core Resources — `main.tf`

- Data sources:
  - AMI: latest Amazon Linux 2023 ARM64 (`al2023-ami-*-arm64`).
  - Availability Zones: first available AZ is used.
  - Cloudflare zone lookup by `cloudflare_zone_name`.
- Locals:
  - `domain`: `staging-<subdomain>.<zone>` for staging; `<subdomain>.<zone>` for prod. Example (staging): `staging-new-cms.avantifellows.org`.
  - `name_prefix`: `nex-gen-cms-<environment>` for tagging/naming.
- Security Group (`aws_security_group.web`):
  - Ingress: 22 from `ssh_cidr`; 80 and 443 from `0.0.0.0/0`.
  - Egress: all.
- Elastic IP (`aws_eip.web`) and association (`aws_eip_association.web`).
- EC2 Instance (`aws_instance.web`):
  - AMI: Amazon Linux 2023 (ARM64), type from `instance_type`.
  - Root volume: 30 GB gp3, encrypted, delete on termination.
  - Injected `user_data` (multi-part MIME) renders `terraform/user-data.sh` with variables.
  - Tags and `create_before_destroy` lifecycles used to minimize downtime.
- DNS (`cloudflare_record.web`):
  - A record to EIP with TTL 300.
  - `proxied = false` to allow Let's Encrypt HTTP-01 challenge to pass unproxied traffic.

### Bootstrap and State — `bootstrap.sh`

One-time helper to create the S3 bucket and DynamoDB table used by the Terraform backend. Validates AWS CLI auth and idempotently creates:

- S3 bucket with versioning, AES256 default encryption, public access blocks.
- DynamoDB table with PAY_PER_REQUEST billing and tags.

### Outputs — `outputs.tf`

Operational outputs after `terraform apply`:

- `instance_id`, `instance_public_ip`, `instance_private_ip`.
- `domain_name`, `application_url` (`https://<domain>`).
- `ssh_command` (suggested SSH command if a key pair is configured; otherwise recommends SSM).
- `cloudflare_record_id`, `cloudflare_zone_id`, `security_group_id`.

## Instance Bootstrapping and Configuration — `user-data.sh`

The user data script runs on every boot (idempotent) and performs:

1) System updates and package install: `git`, `nginx`, `golang`, `certbot`, `python3-certbot-nginx`, `firewalld`.
2) Creates a dedicated `app` user and app directories in `/opt/nex-gen-cms` and `/var/log/nexgencms`.
3) Clones or hard-resets the repo to `repo_branch` at `repo_url`.
4) Writes `.env` in the app directory with `DB_SERVICE_ENDPOINT` and `DB_SERVICE_TOKEN` for the Go app.
5) Builds the app binary to `/opt/nex-gen-cms/nex-gen-cms` (`go build ./cmd`).
6) Creates and enables a `systemd` service `nexgencms` with `WorkingDirectory=/opt/nex-gen-cms` and `ExecStart=/opt/nex-gen-cms/nex-gen-cms`.
7) Configures NGINX:
   - If certificates already exist at `/etc/letsencrypt/live/<domain>/`, configure HTTPS + HSTS and proxy to `127.0.0.1:8080`.
   - Else configure HTTP-only reverse proxy for initial access and ACME challenges.
8) Validates and restarts NGINX.
9) Opens firewall services (http/https/ssh) via `firewalld` when active.
10) Obtains/renews certificates via Certbot for `<domain>`, then rewrites NGINX to HTTPS and reloads.
11) Installs a renewal hook to reload NGINX after auto-renewals; enables the renewal timer.

Logs:

- Setup log: `/var/log/nexgencms-setup.log`
- App service: `systemctl status nexgencms`, `journalctl -u nexgencms -f`
- NGINX: `/var/log/nginx/error.log`, `/var/log/nginx/access.log`
- Certbot: `sudo certbot certificates`

## DNS and TLS

- Domain naming:
  - Staging: `staging-<subdomain>.<cloudflare_zone_name>`.
  - Production: `<subdomain>.<cloudflare_zone_name>` (currently `new-cms.avantifellows.org`).
- Cloudflare A record is created with `proxied = false` to allow ACME HTTP-01 verification. You may enable proxying after certificates are provisioned if desired.
- NGINX terminates TLS and forwards traffic to the Go server at `127.0.0.1:8080`.


## CI/CD — GitHub Actions
### Triggers

- Staging: push to `main` (deploy).
- Production: push to `release` (deploy).
- Pull requests (validation path; see notes below).
- Manual runs (`workflow_dispatch`) for any branch.

### Environment Variables/Inputs

The workflows set TF variables through environment variables (`TF_VAR_*`).

- Staging: `TF_VAR_environment=staging`, `TF_VAR_aws_region=ap-south-1`, `TF_VAR_ssh_cidr=0.0.0.0/0`, `TF_VAR_subdomain=new-cms`, `TF_VAR_instance_type=t4g.small`.
- Secrets (staging): `TF_VAR_CLOUDFLARE_EMAIL`, `TF_VAR_CLOUDFLARE_API_KEY`, `TF_VAR_CLOUDFLARE_ZONE_NAME`, `TF_VAR_LETSENCRYPT_EMAIL`, `TF_VAR_DB_SERVICE_TOKEN_STAGING`, `TF_VAR_KEY_PAIR_NAME`.
- Repo/branch: `TF_VAR_repo_url=https://github.com/avantifellows/nex-gen-cms.git`, `TF_VAR_repo_branch=${{ github.ref_name }}` (current branch).
- App config (staging): `TF_VAR_db_service_endpoint=https://staging-db.avantifellows.org/api/`.

Production-specific overrides:

- Backend key: `terraform init -reconfigure -backend-config="key=nex-gen-cms/prod.tfstate"`.
- `TF_VAR_environment=prod`, `TF_VAR_instance_type=t4g.medium`, `TF_VAR_subdomain=new-cms`.
- `TF_VAR_db_service_endpoint` set to production API, `TF_VAR_db_service_token` from secret.
- Workflow file: `.github/workflows/deploy-prod.yml`.

Required GitHub Secrets (repository settings → Secrets and variables → Actions):

- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` (for Terraform AWS provider).
- `TF_VAR_CLOUDFLARE_EMAIL`, `TF_VAR_CLOUDFLARE_API_KEY`, `TF_VAR_CLOUDFLARE_ZONE_NAME`.
- `TF_VAR_LETSENCRYPT_EMAIL`, `TF_VAR_DB_SERVICE_TOKEN_STAGING`, `TF_VAR_KEY_PAIR_NAME` (optional).
- Production: `TF_VAR_DB_SERVICE_ENDPOINT_PROD`, `TF_VAR_DB_SERVICE_TOKEN_PROD`, `TF_VAR_KEY_PAIR_NAME_PROD` (optional).

Optional: Use Environments named `staging` and `production` with required reviewers to gate applies.

### Job Steps (condensed)

1) Checkout and set up Terraform (`1.5.7`).
2) Configure AWS credentials and region.
3) `terraform fmt -check` (non-blocking), `terraform init -reconfigure`, `terraform validate`, `terraform plan -out=tfplan`.
4) `terraform apply -auto-approve tfplan` and parse output to determine if changes occurred.
5) Capture `instance_id` from outputs.
6) If no infra changes, reboot the EC2 instance to pull latest code at boot.
7) Wait for instance running + status OK.
8) Print `application_url` and `instance_public_ip` outputs; poll the URL up to ~15 minutes for readiness.

Notes:

- On pull requests from forks, secrets are not exposed, so apply will not succeed; treat PR runs as validation only. To enforce approvals, configure a protected Environment and set `environment: staging` on the job.


## Variables and Configuration Files

- `terraform/terraform.tfvars.example`: Template of variables to copy into `terraform.tfvars` for local CLI runs. Do not commit real values.
- `terraform/terraform.tfvars`: Local operator values for staging (sensitive). Should not be committed to VCS; prefer GitHub Secrets + workflow variables for CI.
- `terraform/terraform.prod.tfvars`: Local operator values for production (sensitive). Add to `.gitignore`; prefer `TF_VAR_*` secrets for CI.
- For CI/CD, prefer configuring TF vars via `TF_VAR_*` secrets.

## Operational Runbooks
### First-time Staging Setup

1) Bootstrap remote state:
   - `cd terraform && ./bootstrap.sh`
   - Uncomment the S3 backend in `backend.tf` and run `terraform init -migrate-state`.
2) Configure variables:
   - For local: `cp terraform.tfvars.example terraform.tfvars` and fill in values.
   - For CI: add required GitHub Secrets listed above.
3) Deploy:
   - Local: `terraform plan && terraform apply`.
   - CI: push to `main` or run the workflow manually.
4) Verify:
   - `terraform output`, visit `https://staging-<subdomain>.<zone>`.

### First-time Production Setup (local CLI)

1) Initialize backend with prod key:
   - `cd terraform && terraform init -reconfigure -backend-config="key=nex-gen-cms/prod.tfstate"`
2) Configure variables:
   - Create `terraform/terraform.prod.tfvars` (do not commit), set non-sensitive values; export sensitive `TF_VAR_*` in shell.
3) Deploy:
   - `terraform plan -var-file=terraform.prod.tfvars && terraform apply -var-file=terraform.prod.tfvars`.
4) Verify:
   - `terraform output`, visit `https://<subdomain>.<zone>` (e.g., `https://new-cms.avantifellows.org`).

### Updating Application Code Only

- Push to `main` or run the workflow via `workflow_dispatch`. If Terraform detects no infra drift, the job reboots the instance to pull latest code and restart the app via user data.
- Manual: SSH (or SSM) into the instance and `sudo reboot` to trigger the same flow.

### Accessing the Instance

- Use the `ssh_command` Terraform output if a key pair is configured. Otherwise, use AWS Systems Manager Session Manager.

### Logs & Troubleshooting

- Bootstrapping: `/var/log/nexgencms-setup.log`
- App service: `systemctl status nexgencms`, `journalctl -u nexgencms -f`
- NGINX: `/var/log/nginx/error.log` and `/var/log/nginx/access.log`
- TLS: `sudo certbot certificates`, `sudo nginx -t`

## Security Considerations (current; tighten for prod)

- Restrict `ssh_cidr` to office/VPN IPs; consider removing SSH and using SSM only. Production currently allows `0.0.0.0/0` per note; restrict later.
- Keep Cloudflare `proxied = false` during ACME issuance; optionally enable after issuance for DDoS/WAF in prod.
- Rotate Cloudflare Global API Key; prefer scoped API tokens for production.
- Ensure `db_service_token` and other secrets are only supplied via GitHub Secrets/SSM and never committed.
- Consider: CloudWatch Agent for centralized logs/metrics, IMDSv2 enforcement, least-privilege IAM, OS hardening, automatic security updates.

## Known Trade-offs

- Single EC2 instance per environment; no Auto Scaling, no multi-AZ HA.
- NGINX co-located with app; no managed load balancer.
- In-place code updates via reboot; no blue/green or zero-downtime deployment.
- SSL termination handled on-instance; Cloudflare proxy off by default for ACME.

## Future Production Notes

- Separate VPC/subnets and hardened SG rules.
- Private subnets + public ALB with target group to app instances.
- Auto Scaling Group + rolling/blue-green deploys.
- Observability: CloudWatch Logs/metrics/alarms, distributed tracing, dashboards.
- Secret management via AWS Secrets Manager/SSM Parameter Store.
- WAF, Cloudflare proxying, rate limiting, and stricter firewall.

## Quick References

- Workflows: `.github/workflows/deploy-staging.yml`, `.github/workflows/deploy-prod.yml`
- Terraform: `terraform/main.tf`, `terraform/backend.tf`, `terraform/variables.tf`, `terraform/outputs.tf`, `terraform/user-data.sh`, `terraform/bootstrap.sh`, `terraform/README.md`
- TF Vars: `terraform/terraform.tfvars` (staging), `terraform/terraform.prod.tfvars` (production; gitignored)
- App overview: `context_for_ai/PROJECT_OVERVIEW.md`
