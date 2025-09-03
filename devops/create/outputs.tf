output "route53_zone_id" {
  description = "The Route53 hosted zone ID"
  value       = var.route53_zone_id
}

output "load_balancer_dns" {
  description = "The DNS name of the load balancer"
  value       = aws_lb.app_lb.dns_name
}

output "load_balancer_zone_id" {
  description = "The canonical hosted zone ID of the load balancer"
  value       = aws_lb.app_lb.zone_id
}

output "instance_id" {
  description = "The ID of the EC2 instance"
  value       = aws_instance.app_instance.id
}

output "instance_private_ip" {
  description = "The private IP of the EC2 instance"
  value       = aws_instance.app_instance.private_ip
}

output "target_group_arn" {
  description = "The ARN of the target group"
  value       = aws_lb_target_group.app_tg.arn
}

output "domain_name" {
  description = "The domain of the deployed application"
  value       = var.domain_name
}

output "website_url" {
  description = "The URL of the deployed application"
  value       = "https://${var.domain_name}"
}

output "app_security_group_id" {
  description = "The ID of the security group for the app"
  value       = aws_security_group.app_sg.id
}

output "deployment_id" {
  description = "The ID of the deployment"
  value       = var.deployment_id
}

output "vpc_id" {
  description = "The VPC ID"
  value       = data.aws_vpc.existing.id
}

output "subnet_id" {
  description = "The subnet ID for EC2 instances"
  value       = data.aws_subnet.private1.id
}

output "https_listener_arn" {
  description = "The ARN of the HTTPS listener"
  value       = aws_lb_listener.https.arn
}

output "iam_instance_profile_name" {
  description = "The name of the IAM instance profile"
  value       = aws_iam_instance_profile.ec2_profile.name
}

output "instance_type" {
  description = "The instance type"
  value       = var.instance_type
}

output "ami_id" {
  description = "The AMI ID used for instances"
  value       = data.aws_ami.amazon_linux_2023_arm.id
}

output "app_name" {
  description = "The name of the application"
  value       = var.app_name
}

output "load_balancer_arn" {
  description = "The ARN of the load balancer"
  value       = aws_lb.app_lb.arn
}
