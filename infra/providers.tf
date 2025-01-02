terraform {
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

  required_version = ">= 1.10.2"
}

provider "aws" {
  region = "ap-south-1"
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
