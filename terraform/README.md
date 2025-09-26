# Nex Gen CMS Terraform Deployment

This Terraform configuration deploys the Nex Gen CMS application to AWS with the following architecture:

- **EC2 Instance**: ARM-based Amazon Linux 2023 instance
- **Load Balancer**: NGINX reverse proxy with SSL termination
- **SSL Certificate**: Let's Encrypt certificate via Certbot
- **DNS**: Cloudflare A record pointing to Elastic IP
- **Storage**: S3 + DynamoDB backend for Terraform state

## Prerequisites

1. **AWS CLI** configured with appropriate permissions
2. **Terraform** >= 1.0 installed
3. **Cloudflare account** with API token
4. **Domain** managed by Cloudflare

## Required AWS Permissions

Your AWS credentials need the following permissions:
- EC2: Full access for instances, security groups, EIPs
- S3: Full access for the state bucket
- DynamoDB: Full access for the lock table

## Setup Instructions

### 1. Bootstrap Backend (One-time setup)

First, create the S3 bucket and DynamoDB table for Terraform state:

```bash
cd terraform

# Run the bootstrap script to create backend resources
./bootstrap.sh

# After the script completes successfully:
# 1. Uncomment the backend block in backend.tf
# 2. Initialize terraform with the backend
terraform init -migrate-state
```

**Important**: The backend block in `backend.tf` is commented out by default. After running the bootstrap script, you need to uncomment it before running `terraform init -migrate-state`.

### 2. Configure Variables

```bash
# Copy the example variables file
cp terraform.tfvars.example terraform.tfvars

# Edit terraform.tfvars with your values
vim terraform.tfvars
```

Required variables:
- `cloudflare_email`: Your Cloudflare account email
- `cloudflare_api_key`: Get from Cloudflare Dashboard → My Profile → API Tokens → Global API Key
- `cloudflare_zone_name`: Your domain name (e.g., `example.com`)
- `subdomain`: Your domain (e.g., `example.com` creates `staging.example.com`)
- `letsencrypt_email`: Email for Let's Encrypt notifications
- `repo_url`: Your Git repository URL
- `db_service_endpoint`: Your database service URL
- `db_service_token`: Your database service authentication token

### 3. Deploy Infrastructure

```bash
# Plan the deployment
terraform plan

# Apply the configuration
terraform apply
```

### 4. Verify Deployment

After deployment, check the outputs:

```bash
terraform output
```

Visit your application at the provided URL (e.g., `https://staging.your-domain.com`).

## GitHub Actions Integration

For CI/CD deployment, add these secrets to your GitHub repository:

### Required Secrets
- `AWS_ACCESS_KEY_ID`: AWS access key
- `AWS_SECRET_ACCESS_KEY`: AWS secret key

#### Required TF_VAR Secrets:
- `TF_VAR_cloudflare_email`: Cloudflare account email
- `TF_VAR_cloudflare_api_key`: Cloudflare Global API key
- `TF_VAR_cloudflare_zone_name`: Your domain name (e.g., `avantifellows.org`)
- `TF_VAR_subdomain`: Your app subdomain (e.g., `new-cms`)
- `TF_VAR_letsencrypt_email`: Email for Let's Encrypt certificates
- `TF_VAR_repo_url`: Your Git repository URL
- `TF_VAR_db_service_endpoint`: Database service endpoint URL
- `TF_VAR_db_service_token`: Database service authentication token

#### Optional TF_VAR Secrets (with defaults):
- `TF_VAR_environment`: Environment name (default: `staging`)
- `TF_VAR_aws_region`: AWS region (default: `ap-south-1`)
- `TF_VAR_ssh_cidr`: SSH access CIDR (default: `0.0.0.0/0`)
- `TF_VAR_instance_type`: EC2 instance type (default: `t4g.small`)
- `TF_VAR_key_pair_name`: EC2 key pair name for SSH access

**Note**: `TF_VAR_repo_branch` is automatically set to the current branch name (`${{ github.ref_name }}`) and doesn't need to be configured as a secret.

### Workflow Behavior

The deployment workflow triggers on:
- **Push to main**: Runs plan + apply (actual deployment)
- **Pull requests to main**: Runs plan only (validation)
- **Manual trigger**: Can be run from any branch via workflow_dispatch

### Setting up Approval Requirements

To require approval before deployments, configure **Environment Protection Rules** in your GitHub repository:

1. Go to **Settings** → **Environments** → **Create environment** (name it `staging`)
2. Add **Required reviewers** (team members who must approve)
3. Update the workflow to use this environment:

```yaml
jobs:
  deploy:
    name: Deploy to AWS
    runs-on: ubuntu-latest
    environment: staging  # Add this line
```

This will pause the workflow before the apply step and require approval from designated reviewers.

## Environments

This configuration supports multiple environments using Terraform variables:

### Staging
- Domain: `staging.your-domain.com`
- Instance: `t4g.small`
- Branch: `main`

### Production (Future)
Change these variables for production:
- `environment = "prod"`
- `repo_branch = "main"` or `"prod"`
- `instance_type = "t4g.medium"` or larger
- Restrict `ssh_cidr` to your IP range

## Troubleshooting

### Instance Setup Logs
Check the user data script execution:
```bash
ssh ec2-user@<instance-ip>
sudo tail -f /var/log/nexgencms-setup.log
```

### Application Logs
Check the application service:
```bash
sudo systemctl status nexgencms
sudo journalctl -u nexgencms -f
```

### NGINX Logs
Check web server logs:
```bash
sudo tail -f /var/log/nginx/error.log
sudo tail -f /var/log/nginx/access.log
```

### SSL Certificate Issues
Check certificate status:
```bash
sudo certbot certificates
sudo nginx -t
```

## Maintenance

### Updates
The instance automatically pulls the latest code on every boot. To trigger an update:
```bash
sudo reboot
```

### SSL Certificate Renewal
Certificates auto-renew via systemd timer. Check status:
```bash
sudo systemctl status certbot-renew.timer
sudo certbot renew --dry-run
```

## Security Considerations

1. **SSH Access**: Restrict `ssh_cidr` to your IP range
2. **Key Pairs**: Use EC2 key pairs for SSH access
3. **Secrets**: Never commit sensitive variables to version control
4. **HTTPS**: Always use HTTPS in production
5. **Updates**: Regularly update the instance and application

## Cleanup

To destroy the infrastructure:
```bash
terraform destroy
```

**Note**: This will delete all resources including the EC2 instance and Elastic IP.
