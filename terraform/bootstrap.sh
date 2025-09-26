#!/bin/bash

# Bootstrap script to create S3 bucket and DynamoDB table for Terraform backend
# Run this script before using the main Terraform configuration

set -euo pipefail

BUCKET_NAME="tfstate-nex-gen-cms"
TABLE_NAME="tfstate-nex-gen-cms-locks"
REGION="ap-south-1"

echo "🚀 Creating Terraform backend resources..."

# Check if AWS CLI is configured
if ! aws sts get-caller-identity >/dev/null 2>&1; then
    echo "❌ AWS CLI is not configured. Please run 'aws configure' first."
    exit 1
fi

echo "📦 Creating S3 bucket: $BUCKET_NAME"

# Create S3 bucket
if aws s3api head-bucket --bucket "$BUCKET_NAME" 2>/dev/null; then
    echo "✅ S3 bucket $BUCKET_NAME already exists"
else
    # Create bucket
    aws s3api create-bucket \
        --bucket "$BUCKET_NAME" \
        --region "$REGION" \
        --create-bucket-configuration LocationConstraint="$REGION"
    
    # Enable versioning
    aws s3api put-bucket-versioning \
        --bucket "$BUCKET_NAME" \
        --versioning-configuration Status=Enabled
    
    # Enable encryption
    aws s3api put-bucket-encryption \
        --bucket "$BUCKET_NAME" \
        --server-side-encryption-configuration '{
            "Rules": [
                {
                    "ApplyServerSideEncryptionByDefault": {
                        "SSEAlgorithm": "AES256"
                    }
                }
            ]
        }'
    
    # Block public access
    aws s3api put-public-access-block \
        --bucket "$BUCKET_NAME" \
        --public-access-block-configuration \
        BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
    
    echo "✅ S3 bucket $BUCKET_NAME created successfully"
fi

echo "🗃️  Creating DynamoDB table: $TABLE_NAME"

# Create DynamoDB table
if aws dynamodb describe-table --table-name "$TABLE_NAME" --region "$REGION" >/dev/null 2>&1; then
    echo "✅ DynamoDB table $TABLE_NAME already exists"
else
    aws dynamodb create-table \
        --table-name "$TABLE_NAME" \
        --attribute-definitions AttributeName=LockID,AttributeType=S \
        --key-schema AttributeName=LockID,KeyType=HASH \
        --billing-mode PAY_PER_REQUEST \
        --region "$REGION" \
        --tags Key=Name,Value="Terraform State Lock" Key=Project,Value="nex-gen-cms" Key=Environment,Value="shared" Key=ManagedBy,Value="bootstrap"
    
    echo "⏳ Waiting for DynamoDB table to be active..."
    aws dynamodb wait table-exists --table-name "$TABLE_NAME" --region "$REGION"
    echo "✅ DynamoDB table $TABLE_NAME created successfully"
fi

echo ""
echo "🎉 Backend resources created successfully!"
echo ""
echo "Next steps:"
echo "1. Uncomment the backend block in terraform/backend.tf"
echo "2. Run: terraform init -migrate-state"
echo "3. Proceed with your Terraform deployment"
echo ""
echo "Backend configuration:"
echo "  Bucket: $BUCKET_NAME"
echo "  Table:  $TABLE_NAME"
echo "  Region: $REGION"
