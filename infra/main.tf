resource "aws_instance" "cms" {
  ami           = var.ami_id
  instance_type = var.instance_type
  key_name      = var.key_name

  tags = {
    Name = var.ec2_name
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group_rule" "allow_8080_inbound" {
  type              = "ingress"
  from_port         = 8080
  to_port           = 8080
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = tolist(aws_instance.cms.vpc_security_group_ids)[0]
}

# Associate Elastic IP
data "aws_eip" "existing_eip" {
  public_ip = "13.202.247.153"
}

resource "aws_eip_association" "cms_eip_association" {
  # Don't create this resource for staging, because of elastic ip limit reached to max 10
  count = terraform.workspace == "production" ? 1 : 0

  instance_id   = aws_instance.cms.id
  allocation_id = data.aws_eip.existing_eip.id
}

resource "cloudflare_record" "ec2_domain" {
  zone_id = var.cloudflare_zone_id # Zone ID for our domain in Cloudflare
  name    = var.subdomain          # Subdomain (e.g., 'content' for 'content.domain.com')
  type    = "A"                    # A record
  content = (terraform.workspace == "production"
    ? aws_eip_association.cms_eip_association[0].public_ip # Elastic IP of the EC2 instance for production
  : aws_instance.cms.public_ip)                            # Public IP of the EC2 instance for staging

  ttl     = 1     # Set TTL to automatic
  proxied = false # Set to true if we want to use Cloudflare proxy (orange cloud)
}

# to restart instance if stopped
resource "null_resource" "start_instance" {

  # While running terraform script first time, this resource creation runs forever waiting for public ip of ec2 instance. 
  # To fix that, run start_instance resource only after elastic ip association and domain name mapping completes
  # Hence depends on cloudflare_record
  depends_on = [cloudflare_record.ec2_domain]

  provisioner "local-exec" {
    # interpreter: Forces the provisioner to use Bash instead of the default Windows Command Prompt.
    interpreter = ["bash", "-c"]
    command     = <<EOT
          STATE=$(aws ec2 describe-instances --instance-ids ${aws_instance.cms.id} --region ap-south-1 \
            --query "Reservations[*].Instances[*].State.Name" --output text)

          if [ "$STATE" == "stopped" ]; then
            aws ec2 start-instances --instance-ids ${aws_instance.cms.id} --region ap-south-1
            echo "Instance ${aws_instance.cms.id} started."

            # Wait for the instance to be in 'running' state
            while [ "$(aws ec2 describe-instances --instance-ids ${aws_instance.cms.id} --region ap-south-1 \
              --query "Reservations[*].Instances[*].State.Name" --output text)" != "running" ]; do
              echo "Waiting for instance to start..."
              sleep 5
            done

          else
            echo "Instance ${aws_instance.cms.id} is already in $STATE state. No action taken."
          fi

        EOT
  }

  triggers = {
    always_run = timestamp() # Forces re-execution on each apply
  }
}

resource "null_resource" "run_commands" {
  # Ensures that this runs after start_instance complete
  depends_on = [
    null_resource.start_instance
  ]

  connection {
    type        = "ssh"
    user        = "ec2-user"
    private_key = file("D:/Avanti/cms-key.pem")
    host        = terraform.workspace == "production" ? data.aws_eip.existing_eip.public_ip : aws_instance.cms.public_ip
  }

  provisioner "remote-exec" {
    inline = [
      # Ensure Git is installed
      "if ! command -v git &> /dev/null; then sudo yum install git -y; fi",

      # Update the system
      "sudo yum update -y",

      # Install Nginx if not already installed
      "if ! command -v nginx &> /dev/null; then",
      "  echo 'Nginx not found. Installing...'",
      "  sudo yum install nginx -y",
      "fi",

      # Create Nginx configuration for the domain if not already present
      "if ! grep -q 'server_name ${var.subdomain}.avantifellows.org;' /etc/nginx/conf.d/${var.subdomain}.conf; then",
      "  echo 'Creating Nginx configuration for ${var.subdomain}.avantifellows.org...'",
      "  sudo bash -c 'cat > /etc/nginx/conf.d/${var.subdomain}.conf <<EOF",
      "server {",
      "    listen 80;",
      "    server_name ${var.subdomain}.avantifellows.org;",
      "    location / {",
      "        proxy_pass http://localhost:8080;",
      "        proxy_set_header Host \\$host;",
      "        proxy_set_header X-Real-IP \\$remote_addr;",
      "        proxy_set_header X-Forwarded-For \\$proxy_add_x_forwarded_for;",
      "        proxy_set_header X-Forwarded-Proto \\$scheme;",
      "    }",
      "}",
      "EOF'",
      "  sudo nginx -t && sudo systemctl reload nginx",
      "fi",

      # Install Certbot and its Nginx plugin if not already installed
      "if ! command -v certbot &> /dev/null; then",
      "  echo 'Certbot not found. Installing Certbot and Nginx plugin...'",
      "  sudo yum install certbot python3-certbot-nginx -y",
      "fi",

      # Run Certbot for the domain
      "sudo certbot --nginx -d ${var.subdomain}.avantifellows.org --non-interactive --agree-tos -m ankur@avantifellows.org",
      # Remove blank lines introduced by certbot
      "sudo sed -i '/^$/d' /etc/nginx/conf.d/${var.subdomain}.conf",

      # Start and enable the Nginx service
      "sudo systemctl enable nginx",
      "sudo systemctl start nginx",

      # Define project directory and repository URL
      "PROJECT_DIR='/home/ec2-user/nex-gen-cms'",
      "REPO_URL='https://github.com/avantifellows/nex-gen-cms.git'",

      # Check if the project directory exists
      "if [ ! -d \"$PROJECT_DIR\" ]; then",
      "  echo 'Project directory does not exist. Cloning the repository...'",
      "  git clone $REPO_URL $PROJECT_DIR",
      "else",
      "  echo 'Project directory exists.'",
      "fi",

      # Install Go if not already installed
      "if ! command -v go &> /dev/null; then",
      "  echo 'Go not found. Installing...'",
      "  GO_VERSION=1.23.4",
      "  wget https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz",
      "  sudo tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz",
      "  rm go$GO_VERSION.linux-amd64.tar.gz",
      "  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc",
      "  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile",
      "  source ~/.bashrc || true",
      "  source ~/.profile || true",
      "fi",
    ]
  }

  triggers = {
    always_run = timestamp() # Forces re-execution on each apply
  }
}

data "local_file" "env_file" {
  filename = terraform.workspace == "production" ? "../.env" : "../.env.staging"
}

output "env_checksum" {
  value = filesha256(data.local_file.env_file.filename)
}

resource "null_resource" "upload_env" {
  # Ensures that this runs after run_commands complete
  depends_on = [
    null_resource.run_commands
  ]

  triggers = {
    # Trigger resource update on checksum change
    env_checksum = filesha256(data.local_file.env_file.filename)
  }

  provisioner "file" {
    source      = data.local_file.env_file.filename
    destination = "/home/ec2-user/nex-gen-cms/.env"
  }

  connection {
    host        = terraform.workspace == "production" ? data.aws_eip.existing_eip.public_ip : aws_instance.cms.public_ip
    user        = "ec2-user"
    private_key = file("D:/Avanti/cms-key.pem")
  }
}


