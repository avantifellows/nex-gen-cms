# IAM role for the EC2 instance so the app can read/write cached test PDFs via
# the AWS SDK's default credential chain (instance metadata) — no static
# AWS_ACCESS_KEY_ID/SECRET in .env.
resource "aws_iam_role" "web" {
  name = "${local.name_prefix}-ec2-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })

  tags = {
    Name = "${local.name_prefix}-ec2-role"
  }
}

# Scoped to just this bucket's objects — GetObject also covers HeadObject
# (AWS has no separate HeadObject action).
resource "aws_iam_role_policy" "test_pdfs_access" {
  name = "${local.name_prefix}-test-pdfs-access"
  role = aws_iam_role.web.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["s3:GetObject", "s3:PutObject"]
      Resource = "${aws_s3_bucket.test_pdfs.arn}/*"
    }]
  })
}

resource "aws_iam_instance_profile" "web" {
  name = "${local.name_prefix}-ec2-profile"
  role = aws_iam_role.web.name
}
