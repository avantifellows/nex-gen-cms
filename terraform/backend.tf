# Backend configuration for S3 + DynamoDB state management
# STEP 1: Comment out the backend block below to bootstrap
# STEP 2: After creating S3 bucket and DynamoDB table, uncomment and run terraform init -migrate-state

terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }

  # Backend configuration using partial configuration
  # Initialize with: terraform init -backend-config=backend-staging.hcl (or backend-prod.hcl)
  backend "s3" {
    # Backend config provided via -backend-config flag during init
  }
}

# Provider configurations
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "nex-gen-cms"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

provider "cloudflare" {
  email   = var.cloudflare_email
  api_key = var.cloudflare_api_key
}
