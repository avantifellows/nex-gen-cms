# Infrastructure variables
variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "ap-south-1"
}

variable "environment" {
  description = "Environment name (staging, prod)"
  type        = string
  default     = "staging"
  
  validation {
    condition     = contains(["staging", "prod"], var.environment)
    error_message = "Environment must be either 'staging' or 'prod'."
  }
}

variable "ssh_cidr" {
  description = "CIDR block allowed for SSH access"
  type        = string
  default     = "0.0.0.0/0"  # Restrict this in production
}

# Cloudflare DNS variables
variable "cloudflare_email" {
  description = "Cloudflare account email"
  type        = string
}

variable "cloudflare_api_key" {
  description = "Cloudflare Global API key"
  type        = string
  sensitive   = true
}

variable "cloudflare_zone_name" {
  description = "Cloudflare zone name (domain name)"
  type        = string
}

variable "subdomain" {
  description = "Base subdomain (will create staging.subdomain or prod.subdomain)"
  type        = string
}

variable "letsencrypt_email" {
  description = "Email address for Let's Encrypt certificate registration"
  type        = string
}

# Git repository variables
variable "repo_url" {
  description = "Git repository URL for the application"
  type        = string
  default     = "https://github.com/your-org/nex-gen-cms.git"
}

variable "repo_branch" {
  description = "Git branch to deploy"
  type        = string
  default     = "main"
}

# Application environment variables
variable "db_service_endpoint" {
  description = "Database service endpoint URL"
  type        = string
  sensitive   = true
}

variable "db_service_token" {
  description = "Database service authentication token"
  type        = string
  sensitive   = true
}

# Instance configuration
variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t4g.small"  # ARM-based instance
}

variable "key_pair_name" {
  description = "EC2 Key Pair name for SSH access (optional)"
  type        = string
  default     = null
}
