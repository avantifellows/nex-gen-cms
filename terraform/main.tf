# Data sources - AMI pinned to avoid drift
# Use this pinned AMI instead of most_recent to prevent unnecessary instance replacements
locals {
  pinned_ami_id = "ami-0cbbd6270b5993bd9"  # Amazon Linux 2023 ARM64 - current stable version
}

# Commented out to use pinned AMI above
# data "aws_ami" "amazon_linux" {
#   most_recent = true
#   owners      = ["amazon"]
#
#   filter {
#     name   = "name"
#     values = ["al2023-ami-*-arm64"]
#   }
#
#   filter {
#     name   = "virtualization-type"
#     values = ["hvm"]
#   }
# }

data "aws_availability_zones" "available" {
  state = "available"
}

# Data sources for Cloudflare zone
data "cloudflare_zone" "main" {
  name = var.cloudflare_zone_name
}

# Local values
locals {
  domain      = var.environment == "prod" ? "${var.subdomain}.${var.cloudflare_zone_name}" : "${var.environment}-${var.subdomain}.${var.cloudflare_zone_name}"
  name_prefix = "nex-gen-cms-${var.environment}"
}

# Security Group
resource "aws_security_group" "web" {
  name_prefix = "${local.name_prefix}-"
  description = "Security group for nex-gen-cms web server"

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_cidr]
  }

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${local.name_prefix}-sg"
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Elastic IP
resource "aws_eip" "web" {
  domain = "vpc"

  tags = {
    Name = "${local.name_prefix}-eip"
  }
}

# User data template
locals {
  user_data = <<-EOF
Content-Type: multipart/mixed; boundary="//"
MIME-Version: 1.0

--//
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config.txt"

#cloud-config
cloud_final_modules:
- [scripts-user, always]

--//
Content-Type: text/x-shellscript; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="userdata.txt"

${templatefile("${path.module}/user-data.sh", {
  domain              = local.domain
  repo_url            = var.repo_url
  repo_branch         = var.repo_branch
  db_service_endpoint = var.db_service_endpoint
  db_service_token    = var.db_service_token
  cms_username        = var.cms_username
  cms_password        = var.cms_password
  letsencrypt_email   = var.letsencrypt_email
})}
--//
EOF
}

# EC2 Instance
resource "aws_instance" "web" {
  ami                    = local.pinned_ami_id
  instance_type          = var.instance_type
  key_name               = var.key_pair_name
  vpc_security_group_ids = [aws_security_group.web.id]
  availability_zone      = data.aws_availability_zones.available.names[0]

  user_data = local.user_data

  root_block_device {
    volume_type           = "gp3"
    volume_size           = 30
    delete_on_termination = true
    encrypted             = true
  }

  tags = {
    Name = "${local.name_prefix}-instance"
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Associate Elastic IP with Instance
resource "aws_eip_association" "web" {
  instance_id   = aws_instance.web.id
  allocation_id = aws_eip.web.id
}

# Cloudflare DNS Record
resource "cloudflare_record" "web" {
  zone_id = data.cloudflare_zone.main.id
  name    = var.environment == "prod" ? var.subdomain : "${var.environment}-${var.subdomain}"
  content = aws_eip.web.public_ip
  type    = "A"
  ttl     = 300
  proxied = false # Set to false initially for Let's Encrypt HTTP-01 challenge

  comment = "Terraform managed - ${var.environment} environment for nex-gen-cms"
}
