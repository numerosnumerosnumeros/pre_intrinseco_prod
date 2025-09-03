#!/bin/bash

# Call from root directory: ./devops/create/create.sh

set -e

echo "=========================================="
echo "🚀 Starting deployment..."
START_TIME=$(date +%s)

echo "📂 Setting deployment variables..."
ROOT_DIR=$(pwd)
DEPLOYMENT_ID="v$(date +%Y%m%d%H%M%S)"
APP_NAME="nodo-mono"
AWS_REGION="eu-central-1"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query "Account" --output text)

# Compile
$ROOT_DIR/devops/build.sh --clean --prod

# Clean up
cd "${ROOT_DIR}/devops/create"
echo "🔧 Starting AWS CLI commands..."
rm -rf terraform.tfstate .terraform .terraform.lock.hcl
./cleanup.sh

# Github
echo "🗂️ Commiting to Github..."
git add -A
git commit --allow-empty -m "🚀 Deploy ${DEPLOYMENT_ID}"
git push

# AWS CLI
echo "🔧 Initializing Terraform..."
terraform init

echo "🏗️ Creating infrastructure with deployment ID: '${DEPLOYMENT_ID}'..."
terraform apply -auto-approve \
    -var="deployment_id=${DEPLOYMENT_ID}" \
    -var="account_id=${AWS_ACCOUNT_ID}" \
    -var="aws_region=${AWS_REGION}" \
    -var="app_name=${APP_NAME}" \
    -var="project_root=${ROOT_DIR}"

echo "💾 Saving deployment outputs..."
terraform output | sed 's/ = /=/' | sed 's/"//g' >${ROOT_DIR}/devops/update/outputs.txt

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
echo "✅ Deployment ${DEPLOYMENT_ID} completed successfully"
echo "⏱️ Deployment time: $((DURATION / 60)) min and $((DURATION % 60)) sec"
echo "=========================================="
