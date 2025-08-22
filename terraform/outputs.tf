# Output values for the deployment
output "instance_id" {
  description = "ID of the EC2 instance"
  value       = aws_instance.web.id
}

output "instance_public_ip" {
  description = "Public IP address of the EC2 instance"
  value       = aws_eip.web.public_ip
}

output "instance_private_ip" {
  description = "Private IP address of the EC2 instance"
  value       = aws_instance.web.private_ip
}

output "domain_name" {
  description = "Full domain name for the application"
  value       = local.domain
}

output "application_url" {
  description = "HTTPS URL for the application"
  value       = "https://${local.domain}"
}

output "ssh_command" {
  description = "SSH command to connect to the instance (if key pair is configured)"
  value       = var.key_pair_name != null ? "ssh -i ~/.ssh/${var.key_pair_name}.pem ec2-user@${aws_eip.web.public_ip}" : "Use AWS Systems Manager Session Manager to connect"
}

output "cloudflare_record_id" {
  description = "Cloudflare DNS record ID"
  value       = cloudflare_record.web.id
}

output "cloudflare_zone_id" {
  description = "Cloudflare zone ID (looked up from zone name)"
  value       = data.cloudflare_zone.main.id
}

output "security_group_id" {
  description = "ID of the security group"
  value       = aws_security_group.web.id
}
