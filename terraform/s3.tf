# S3 bucket for cached question-paper/answer-sheet/combined PDFs.
# Account ID is prefixed so the bucket name is globally unique and clearly ours.
resource "aws_s3_bucket" "test_pdfs" {
  bucket = "111766607077-${local.name_prefix}-test-pdfs"

  tags = {
    Name = "${local.name_prefix}-test-pdfs"
  }
}

resource "aws_s3_bucket_public_access_block" "test_pdfs" {
  bucket = aws_s3_bucket.test_pdfs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "test_pdfs" {
  bucket = aws_s3_bucket.test_pdfs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}
