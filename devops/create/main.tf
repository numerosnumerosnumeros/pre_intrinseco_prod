provider "aws" {
  region = var.aws_region
}

#
##
###
####
##### VPC
data "aws_vpc" "existing" {
  id = var.vpc_id
}

data "aws_subnet" "public1" {
  id = var.public_subnet_1_id
}

data "aws_subnet" "public2" {
  id = var.public_subnet_2_id
}

data "aws_subnet" "private1" {
  id = var.private_subnet_1_id
}

data "aws_subnet" "private2" {
  id = var.private_subnet_2_id
}

#
##
###
####
##### CloudWatch log group
resource "aws_cloudwatch_log_group" "app_logs" {
  name              = "/ec2/${var.app_name}-${var.deployment_id}"
  retention_in_days = 180

  tags = {
    Name = "${var.app_name}-logs-${var.deployment_id}"
    App  = var.app_name
  }
}

#
##
###
####
##### IAM
resource "aws_iam_role" "ec2_role" {
  name = "${var.app_name}-ec2-role-${var.deployment_id}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name = "${var.app_name}-ec2-role-${var.deployment_id}"
    App  = var.app_name
  }
}

resource "aws_iam_policy" "cloudwatch_policy" {
  name        = "${var.app_name}-cloudwatch-access-${var.deployment_id}"
  description = "Allow writing to CloudWatch logs"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams",
          "logs:DescribeLogGroups"
        ]
        Resource = [
          "arn:aws:logs:${var.aws_region}:${var.account_id}:log-group:/ec2/${var.app_name}*",
          "arn:aws:logs:${var.aws_region}:${var.account_id}:log-group:/ec2/${var.app_name}*:*"
        ]
      }
    ]
  })
}

resource "aws_iam_policy" "s3_access_policy" {
  name        = "${var.app_name}-s3-access-${var.deployment_id}"
  description = "Allow S3 put operations"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          "arn:aws:s3:::${var.bucket_name}",
          "arn:aws:s3:::${var.bucket_name}/*",
          "arn:aws:s3:::compiled-prod-nodofinance",
          "arn:aws:s3:::compiled-prod-nodofinance/*"
        ]
      }
    ]
  })
}

resource "aws_iam_policy" "cognito_access_policy" {
  name        = "${var.app_name}-cognito-access-${var.deployment_id}"
  description = "Allow specific Cognito operations"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cognito-idp:InitiateAuth",
          "cognito-idp:ForgotPassword",
          "cognito-idp:AdminUpdateUserAttributes",
          "cognito-idp:ConfirmSignUp",
          "cognito-idp:ResendConfirmationCode",
          "cognito-idp:ConfirmForgotPassword",
          "cognito-idp:SignUp",
          "cognito-idp:ListUsers"
        ]
        Resource = "arn:aws:cognito-idp:${var.aws_region}:${var.account_id}:userpool/${var.user_pool_id}"
      }
    ]
  })
}

resource "aws_iam_policy" "ssm_policy" {
  name        = "${var.app_name}-ssm-access-${var.deployment_id}"
  description = "Allow access to SSM Parameter Store"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParametersByPath",
          "ssm:GetParameters",
          "ssm:GetParameter"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:${var.account_id}:parameter/*"
      }
    ]
  })
}

resource "aws_iam_policy" "dynamodb_access_policy" {
  name        = "${var.app_name}-dynamodb-access-${var.deployment_id}"
  description = "Allow DynamoDB operations"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query",
          "dynamodb:TransactWriteItems",
          "dynamodb:Scan",
          "dynamodb:BatchGetItem",
          "dynamodb:BatchWriteItem"
        ]
        Resource = [
          "arn:aws:dynamodb:${var.aws_region}:${var.account_id}:table/nodofinance_table",
          "arn:aws:dynamodb:${var.aws_region}:${var.account_id}:table/nodofinance_table/index/*"
        ]
      }
    ]
  })
}


resource "aws_iam_role_policy_attachment" "cloudwatch_access" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = aws_iam_policy.cloudwatch_policy.arn
}

resource "aws_iam_role_policy_attachment" "s3_access" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = aws_iam_policy.s3_access_policy.arn
}

resource "aws_iam_role_policy_attachment" "cognito_access" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = aws_iam_policy.cognito_access_policy.arn
}

resource "aws_iam_role_policy_attachment" "ssm_access" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = aws_iam_policy.ssm_policy.arn
}

resource "aws_iam_role_policy_attachment" "cloudwatch_agent_policy" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
}

resource "aws_iam_role_policy_attachment" "dynamodb_access" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = aws_iam_policy.dynamodb_access_policy.arn
}

resource "aws_iam_instance_profile" "ec2_profile" {
  name = "${var.app_name}-ec2-profile-${var.deployment_id}"
  role = aws_iam_role.ec2_role.name
}

#
##
###
####
##### ALB security group
resource "aws_security_group" "alb_sg" {
  name        = "${var.app_name}-alb-sg-${var.deployment_id}"
  description = "Security group for ALB"
  vpc_id      = data.aws_vpc.existing.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.app_name}-alb-sg-${var.deployment_id}"
    App  = var.app_name
  }
}

#
##
###
####
##### EC2 security group
resource "aws_security_group" "app_sg" {
  name        = "${var.app_name}-sg-${var.deployment_id}"
  description = "Allow inbound traffic for EC2"
  vpc_id      = data.aws_vpc.existing.id

  ingress {
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    security_groups = [aws_security_group.alb_sg.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.app_name}-sg-${var.deployment_id}"
    App  = var.app_name
  }

  lifecycle {
    create_before_destroy = true
  }
}

#
##
###
####
##### EC2 Instance
data "aws_ami" "amazon_linux_2023_arm" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-minimal-*-arm64"]
  }

  filter {
    name   = "state"
    values = ["available"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  filter {
    name   = "architecture"
    values = ["arm64"]
  }
}

resource "aws_instance" "app_instance" {
  ami           = data.aws_ami.amazon_linux_2023_arm.id
  instance_type = var.instance_type
  #   subnet_id = data.aws_subnet.public1.id
  subnet_id              = data.aws_subnet.private1.id
  vpc_security_group_ids = [aws_security_group.app_sg.id]
  iam_instance_profile   = aws_iam_instance_profile.ec2_profile.name
  #   key_name = var.key_name

  user_data = file("${path.module}/ec2.sh")

  root_block_device {
    volume_size = 20
    volume_type = "gp3"
    encrypted   = true
  }

  tags = {
    Name = "${var.app_name}-instance-${var.deployment_id}"
    App  = var.app_name
  }
}

#
##
###
####
##### ACM
data "aws_acm_certificate" "existing" {
  domain   = var.domain_name
  statuses = ["ISSUED"]
}

#
##
###
####
##### ALB
resource "aws_lb" "app_lb" {
  name               = "${var.app_name}-alb-${var.deployment_id}"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = [data.aws_subnet.public1.id, data.aws_subnet.public2.id]

  enable_deletion_protection = false

  tags = {
    Name = "${var.app_name}-alb-${var.deployment_id}"
    App  = var.app_name
  }
}

resource "aws_wafv2_web_acl_association" "app_waf_association" {
  resource_arn = aws_lb.app_lb.arn
  web_acl_arn  = var.waf_web_acl
}

resource "aws_lb_target_group" "app_tg" {
  name     = "${var.app_name}-tg-${var.deployment_id}"
  port     = 80
  protocol = "HTTP"
  vpc_id   = data.aws_vpc.existing.id

  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 3
    interval            = 30
    path                = "/health"
    port                = "traffic-port"
  }

  tags = {
    Name = "${var.app_name}-tg-${var.deployment_id}"
    App  = var.app_name
  }
}

resource "aws_lb_target_group_attachment" "app_tg_attachment" {
  target_group_arn = aws_lb_target_group.app_tg.arn
  target_id        = aws_instance.app_instance.id
  port             = 80
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.app_lb.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }

  tags = {
    Name = "${var.app_name}-http-listener-${var.deployment_id}"
    App  = var.app_name
  }
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.app_lb.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = data.aws_acm_certificate.existing.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app_tg.arn
  }

  tags = {
    Name = "${var.app_name}-https-listener-${var.deployment_id}"
    App  = var.app_name
  }
}


#
##
###
####
##### Route53
data "aws_route53_zone" "selected" {
  zone_id      = var.route53_zone_id
  private_zone = false
}

resource "aws_route53_record" "app_dns" {
  zone_id = data.aws_route53_zone.selected.zone_id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = aws_lb.app_lb.dns_name
    zone_id                = aws_lb.app_lb.zone_id
    evaluate_target_health = true
  }
}
