#!/bin/bash
set -e

# Call from root directory: ./devops/update/update.sh

START_TIME=$(date +%s)
NEW_DEPLOYMENT_ID="v$(date +%Y%m%d%H%M%S)"
echo "Starting update with deployment ID: ${NEW_DEPLOYMENT_ID}"

ROOT_DIR=$(pwd)

cleanup() {
    echo "Error occurred! Cleaning up resources..."

    if [ -n "${TEST_LISTENER_ARN:-}" ]; then
        echo "Removing test listener..."
        aws elbv2 delete-listener --listener-arn ${TEST_LISTENER_ARN} || true
    fi

    if [ -n "${NEW_TG_ARN:-}" ]; then
        echo "Removing target group..."
        aws elbv2 delete-target-group --target-group-arn ${NEW_TG_ARN} || true
    fi

    if [ -n "${NEW_INSTANCE_ID:-}" ]; then
        echo "Terminating EC2 instance..."
        aws ec2 terminate-instances --instance-ids ${NEW_INSTANCE_ID} || true
    fi

    echo "Cleanup completed"
    exit 1
}
trap cleanup ERR

OUTPUTS_FILE="${ROOT_DIR}/devops/update/outputs.txt"
if [ ! -f "${OUTPUTS_FILE}" ]; then
    echo "Error: Outputs file not found at ${OUTPUTS_FILE}"
    exit 1
fi

echo "Compiling..."
$ROOT_DIR/devops/build.sh --clean --prod

echo "Reading configuration values..."
WEBSITE_URL=$(grep "^website_url=" ${OUTPUTS_FILE} | cut -d= -f2)
APP_SG_ID=$(grep "^app_security_group_id=" ${OUTPUTS_FILE} | cut -d= -f2)
OLD_INSTANCE_ID=$(grep "^instance_id=" ${OUTPUTS_FILE} | cut -d= -f2)
OLD_TG_ARN=$(grep "^target_group_arn=" ${OUTPUTS_FILE} | cut -d= -f2)
DEPLOYMENT_ID=$(grep "^deployment_id=" ${OUTPUTS_FILE} | cut -d= -f2)
VPC_ID=$(grep "^vpc_id=" ${OUTPUTS_FILE} | cut -d= -f2)
SUBNET_ID=$(grep "^subnet_id=" ${OUTPUTS_FILE} | cut -d= -f2)
LISTENER_ARN=$(grep "^https_listener_arn=" ${OUTPUTS_FILE} | cut -d= -f2)
IAM_PROFILE=$(grep "^iam_instance_profile_name=" ${OUTPUTS_FILE} | cut -d= -f2)
INSTANCE_TYPE=$(grep "^instance_type=" ${OUTPUTS_FILE} | cut -d= -f2)
AMI_ID=$(grep "^ami_id=" ${OUTPUTS_FILE} | cut -d= -f2)
APP_NAME=$(grep "^app_name=" ${OUTPUTS_FILE} | cut -d= -f2 || echo "app")

for VAR in WEBSITE_URL APP_SG_ID OLD_INSTANCE_ID OLD_TG_ARN VPC_ID SUBNET_ID LISTENER_ARN IAM_PROFILE INSTANCE_TYPE AMI_ID; do
    if [ -z "${!VAR}" ]; then
        echo "Error: Required value ${VAR} not found in outputs file"
        exit 1
    fi
done

echo "🗂️ Commiting to Github..."
git add -A
git commit --allow-empty -m "🚀 Update ${NEW_DEPLOYMENT_ID}"
git push

#
##
###
####
##### Launch new EC2 instance
echo "Launching new EC2 instance..."
NEW_INSTANCE_ID=$(aws ec2 run-instances \
    --image-id ${AMI_ID} \
    --instance-type ${INSTANCE_TYPE} \
    --subnet-id ${SUBNET_ID} \
    --security-group-ids ${APP_SG_ID} \
    --iam-instance-profile Name=${IAM_PROFILE} \
    --block-device-mappings "DeviceName=/dev/xvda,Ebs={VolumeSize=20,VolumeType=gp3,Encrypted=true}" \
    --user-data file://${ROOT_DIR}/devops/create/ec2.sh \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${APP_NAME}-instance-${NEW_DEPLOYMENT_ID}},{Key=App,Value=${APP_NAME}}]" \
    --output text \
    --query 'Instances[0].InstanceId')

echo "New instance launched: ${NEW_INSTANCE_ID}"

echo "Waiting for instance to be in running state..."
aws ec2 wait instance-running --instance-ids ${NEW_INSTANCE_ID}

#
##
###
####
##### Create new target group
TG_NAME="${APP_NAME}-tg-${NEW_DEPLOYMENT_ID}"

echo "Creating new target group..."
NEW_TG_ARN=$(aws elbv2 create-target-group \
    --name "${TG_NAME}" \
    --protocol HTTP \
    --port 80 \
    --vpc-id ${VPC_ID} \
    --health-check-path /health \
    --health-check-interval-seconds 30 \
    --health-check-timeout-seconds 3 \
    --healthy-threshold-count 2 \
    --unhealthy-threshold-count 2 \
    --output text \
    --query 'TargetGroups[0].TargetGroupArn')

echo "New target group created: ${NEW_TG_ARN}"

# Tag the target group
aws elbv2 add-tags \
    --resource-arns ${NEW_TG_ARN} \
    --tags "Key=Name,Value=${APP_NAME}-tg-${NEW_DEPLOYMENT_ID}" "Key=App,Value=${APP_NAME}"

#
##
###
####
##### Register new instance with target group
echo "Registering instance with target group..."
aws elbv2 register-targets \
    --target-group-arn ${NEW_TG_ARN} \
    --targets Id=${NEW_INSTANCE_ID}

#
##
###
####
##### Wait for the instance to pass health checks (with timeout)
echo "Getting ALB ARN from listener..."
ALB_ARN=$(aws elbv2 describe-listeners \
    --listener-arns ${LISTENER_ARN} \
    --query 'Listeners[0].LoadBalancerArn' \
    --output text)

echo "ALB ARN: ${ALB_ARN}"

echo "Creating temporary test listener for health checks..."
TEST_LISTENER_ARN=$(aws elbv2 create-listener \
    --load-balancer-arn ${ALB_ARN} \
    --protocol HTTP \
    --port 8080 \
    --default-actions Type=forward,TargetGroupArn=${NEW_TG_ARN} \
    --output text \
    --query 'Listeners[0].ListenerArn')

echo "Waiting for health checks to pass (timeout: 5 minutes)..."
MAX_ATTEMPTS=30
attempts=0

while [ ${attempts} -lt ${MAX_ATTEMPTS} ]; do
    HEALTH_STATUS=$(aws elbv2 describe-target-health \
        --target-group-arn ${NEW_TG_ARN} \
        --targets Id=${NEW_INSTANCE_ID} \
        --query 'TargetHealthDescriptions[0].TargetHealth.State' \
        --output text)

    echo "Health status: ${HEALTH_STATUS}"

    if [ "${HEALTH_STATUS}" == "healthy" ]; then
        echo "Instance is healthy!"
        break
    elif [ ${attempts} -eq $((MAX_ATTEMPTS - 1)) ]; then
        echo "Error: Instance failed to become healthy within timeout period"
        echo "Cleaning up resources..."
        aws elbv2 delete-listener --listener-arn ${TEST_LISTENER_ARN}
        aws elbv2 delete-target-group --target-group-arn ${NEW_TG_ARN}
        aws ec2 terminate-instances --instance-ids ${NEW_INSTANCE_ID}
        exit 1
    else
        echo "Instance is ${HEALTH_STATUS}. Checking again in 10 seconds... \(Attempt ${attempts}/${MAX_ATTEMPTS}\)"
        sleep 10
        attempts=$((attempts + 1))
    fi
done

#
##
###
####
##### Modify the HTTPS listener to use the new target group
echo "Switching traffic to new instance..."
aws elbv2 modify-listener \
    --listener-arn ${LISTENER_ARN} \
    --default-actions Type=forward,TargetGroupArn=${NEW_TG_ARN}

echo "Traffic successfully switched to new instance!"
echo "New deployment is live at ${WEBSITE_URL}"

echo "Removing temporary test listener..."
aws elbv2 delete-listener --listener-arn ${TEST_LISTENER_ARN}

DEPLOY_TIME=$(date +%s)
DEPLOY_DURATION=$((DEPLOY_TIME - START_TIME))
echo "⏱️ Update time: $((DEPLOY_DURATION / 60)) min and $((DEPLOY_DURATION % 60)) sec"

#
#
##
###
####
##### Clean up old resources
echo "Waiting 60 seconds before cleanup to ensure stable operation..."
sleep 60
echo "Deregistering old instance from target group..."
aws elbv2 deregister-targets \
    --target-group-arn ${OLD_TG_ARN} \
    --targets Id=${OLD_INSTANCE_ID}

echo "Terminating old instance..."
aws ec2 terminate-instances --instance-ids ${OLD_INSTANCE_ID}

echo "Waiting for instance termination..."
aws ec2 wait instance-terminated --instance-ids ${OLD_INSTANCE_ID}

echo "Deleting old target group..."
aws elbv2 delete-target-group --target-group-arn ${OLD_TG_ARN}

echo "Update completed successfully!"
echo "New instance ID: ${NEW_INSTANCE_ID}"
echo "New target group ARN: ${NEW_TG_ARN}"
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
echo "⏱️ Total time: $((DURATION / 60)) min and $((DURATION % 60)) sec"

#
##
###
####
##### Update outputs.txt
sed -i '' "s|^instance_id=.*|instance_id=${NEW_INSTANCE_ID}|" ${OUTPUTS_FILE}
sed -i '' "s|^target_group_arn=.*|target_group_arn=${NEW_TG_ARN}|" ${OUTPUTS_FILE}
sed -i '' "s|^deployment_id=.*|deployment_id=${NEW_DEPLOYMENT_ID}|" ${OUTPUTS_FILE}

echo "Outputs file updated with new instance and target group information."
