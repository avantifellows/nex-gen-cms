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

  # Comment out this backend block for initial bootstrap
  # Uncomment after creating the S3 bucket and DynamoDB table
  backend "s3" {
    bucket         = "tfstate-nex-gen-cms"
    key            = "nex-gen-cms/staging.tfstate"
    region         = "ap-south-1"
    dynamodb_table = "tfstate-nex-gen-cms-locks"
    encrypt        = true
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
