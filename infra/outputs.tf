output "instance_id" {
  value = aws_instance.cms.id
  description = "The ID of the EC2 instance"
}

output "public_ip" {
  value = aws_instance.cms.public_ip
  description = "The public IP address of the EC2 instance"
}

output "public_dns" {
  value = aws_instance.cms.public_dns
  description = "The public DNS name of the EC2 instance"
}
