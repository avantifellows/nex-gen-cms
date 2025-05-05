variable "instance_type" {
  description = "The type of EC2 instance to use"
  type        = string
}

variable "ami_id" {
  description = "AMI ID for the EC2 instance"
  type        = string
}

variable "key_name" {
  description = "The name of the SSH key pair"
  type        = string
  sensitive   = true
}

variable "cloudflare_api_token" {
  description = "API token for managing DNS in Cloudflare"
  type        = string
  sensitive   = true
}

variable "cloudflare_zone_id" {
  description = "Cloudflare Zone ID"
  type        = string
}

variable "ec2_name" {
  description = "ec2 instance name for the application (CMS or staging-CMS)"
  type        = string
}

variable "subdomain" {
  description = "Subdomain name for the application (content or staging-content)"
  type        = string
}