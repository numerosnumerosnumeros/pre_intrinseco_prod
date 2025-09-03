#!/bin/bash

#
##
###
####
##### EC2
instances=$(aws ec2 describe-instances --filters "Name=tag:Name,Values=nodo-mono*" --query "Reservations[*].Instances[*].{InstanceId:InstanceId,Name:Tags[?Key=='Name'].Value|[0]}" --output json)

echo "Found the following instances to terminate:"
echo "$instances" | jq -r '.[][].Name'
echo "---"

# Stop and terminate each instance
echo "$instances" | jq -c '.[][]' | while read -r instance; do
    instance_id=$(echo "$instance" | jq -r '.InstanceId')
    instance_name=$(echo "$instance" | jq -r '.Name')

    echo "Processing instance: $instance_name ($instance_id)"

    # Stop the instance
    echo "  Stopping instance..."
    aws ec2 stop-instances --instance-ids "$instance_id"

    # Wait for the instance to stop
    echo "  Waiting for instance to stop..."
    aws ec2 wait instance-stopped --instance-ids "$instance_id"

    # Terminate the instance
    echo "  Terminating instance..."
    aws ec2 terminate-instances --instance-ids "$instance_id"

    echo "✓ Terminated: $instance_name ($instance_id)"
done

echo "All 'nodo-mono' EC2 instances have been terminated"

#
##
###
####
##### Route53
RECORD_TYPE="A"
HOSTED_ZONE_ID="Z00130993A05X2HOQW7VA"

delete_dns_record() {
    local domain=$1

    echo "Finding A record for $domain..."
    RECORD_SET=$(aws route53 list-resource-record-sets \
        --hosted-zone-id "$HOSTED_ZONE_ID" \
        --query "ResourceRecordSets[?Name=='$domain.' && Type=='$RECORD_TYPE']" \
        --output json)

    RECORD_COUNT=$(echo "$RECORD_SET" | jq 'length')
    if [ "$RECORD_COUNT" -eq 0 ]; then
        echo "No $RECORD_TYPE record found for $domain"
        return
    fi

    echo "Found $RECORD_TYPE record for $domain:"
    echo "$RECORD_SET" | jq '.'

    TEMP_FILE=$(mktemp)
    cat >"$TEMP_FILE" <<EOF
{
  "Changes": [
    {
      "Action": "DELETE",
      "ResourceRecordSet": $(echo "$RECORD_SET" | jq '.[0]')
    }
  ]
}
EOF

    echo "Deleting A record for $domain..."
    CHANGE_ID=$(aws route53 change-resource-record-sets \
        --hosted-zone-id "$HOSTED_ZONE_ID" \
        --change-batch file://"$TEMP_FILE" \
        --query "ChangeInfo.Id" \
        --output text)

    rm "$TEMP_FILE"

    echo "Delete request submitted with change ID: $CHANGE_ID"
    echo "Waiting for change to propagate..."

    aws route53 wait resource-record-sets-changed --id "$CHANGE_ID"

    echo "✓ Successfully deleted A record for $domain"
}

# Delete both records
echo "Using hosted zone ID: $HOSTED_ZONE_ID"

delete_dns_record "nodo.finance"

#
##
###
####
##### IAM Policies
policies=$(aws iam list-policies --scope Local --query "Policies[?starts_with(PolicyName, 'nodo-mono')].{ARN:Arn,Name:PolicyName}" --output json)

echo "Found the following policies to remove:"
echo "$policies" | jq -r '.[].Name'
echo "---"

echo "$policies" | jq -c '.[]' | while read -r policy; do
    policy_arn=$(echo "$policy" | jq -r '.ARN')
    policy_name=$(echo "$policy" | jq -r '.Name')

    echo "Processing policy: $policy_name"

    attached_entities=$(aws iam list-entities-for-policy --policy-arn "$policy_arn")

    for group_name in $(echo "$attached_entities" | jq -r '.PolicyGroups[].GroupName'); do
        echo "  Detaching from group: $group_name"
        aws iam detach-group-policy --group-name "$group_name" --policy-arn "$policy_arn"
    done

    for user_name in $(echo "$attached_entities" | jq -r '.PolicyUsers[].UserName'); do
        echo "  Detaching from user: $user_name"
        aws iam detach-user-policy --user-name "$user_name" --policy-arn "$policy_arn"
    done

    for role_name in $(echo "$attached_entities" | jq -r '.PolicyRoles[].RoleName'); do
        echo "  Detaching from role: $role_name"
        aws iam detach-role-policy --role-name "$role_name" --policy-arn "$policy_arn"
    done

    versions=$(aws iam list-policy-versions --policy-arn "$policy_arn" --query "Versions[?!IsDefaultVersion].VersionId" --output json)
    for version_id in $(echo "$versions" | jq -r '.[]'); do
        echo "  Deleting version: $version_id"
        aws iam delete-policy-version --policy-arn "$policy_arn" --version-id "$version_id"
    done

    echo "  Deleting policy: $policy_name"
    aws iam delete-policy --policy-arn "$policy_arn"
    echo "✓ Removed: $policy_name"
done

echo "All 'nodo-mono' policies have been removed"

#
##
###
####
##### ALB
echo ""
echo "Looking for Application Load Balancers with name starting with 'nodo-mono'..."

load_balancers=$(aws elbv2 describe-load-balancers --query "LoadBalancers[?starts_with(LoadBalancerName, 'nodo-mono')].{ARN:LoadBalancerArn,Name:LoadBalancerName}" --output json)

echo "Found the following ALBs to remove:"
echo "$load_balancers" | jq -r '.[].Name'
echo "---"

echo "$load_balancers" | jq -c '.[]' | while read -r alb; do
    alb_arn=$(echo "$alb" | jq -r '.ARN')
    alb_name=$(echo "$alb" | jq -r '.Name')

    echo "Processing ALB: $alb_name"

    echo "  Deleting ALB..."
    aws elbv2 delete-load-balancer --load-balancer-arn "$alb_arn"

    echo "✓ Removed ALB: $alb_name"
done

echo "All 'nodo-mono' Application Load Balancers have been removed"

echo ""
echo "Looking for Target Groups with name starting with 'nodo-mono'..."

#
##
###
####
##### CloudWatch
echo ""
echo "Looking for CloudWatch Log Groups with name starting with '/ec2/nodo-mono'..."

log_groups=$(aws logs describe-log-groups --log-group-name-prefix "/ec2/nodo-mono" --query "logGroups[*].{Name:logGroupName}" --output json)

echo "Found the following Log Groups to remove:"
echo "$log_groups" | jq -r '.[].Name'
echo "---"

echo "$log_groups" | jq -c '.[]' | while read -r log_group; do
    log_group_name=$(echo "$log_group" | jq -r '.Name')

    echo "Processing Log Group: $log_group_name"

    echo "  Deleting Log Group..."
    aws logs delete-log-group --log-group-name "$log_group_name"

    echo "✓ Removed Log Group: $log_group_name"
done

echo "All '/ec2/nodo-mono' CloudWatch Log Groups have been removed"

#
##
###
####
##### IAM Roles
echo ""
echo "Looking for IAM Roles with name starting with 'nodo-mono'..."

roles=$(aws iam list-roles --query "Roles[?starts_with(RoleName, 'nodo-mono')].{Name:RoleName}" --output json)

echo "Found the following Roles to remove:"
echo "$roles" | jq -r '.[].Name'
echo "---"

echo "$roles" | jq -c '.[]' | while read -r role; do
    role_name=$(echo "$role" | jq -r '.Name')

    echo "Processing Role: $role_name"

    attached_policies=$(aws iam list-attached-role-policies --role-name "$role_name" --query "AttachedPolicies[*].{ARN:PolicyArn}" --output json)

    for policy in $(echo "$attached_policies" | jq -c '.[]'); do
        policy_arn=$(echo "$policy" | jq -r '.ARN')
        echo "  Detaching policy: $policy_arn"
        aws iam detach-role-policy --role-name "$role_name" --policy-arn "$policy_arn"
    done

    inline_policies=$(aws iam list-role-policies --role-name "$role_name" --query "PolicyNames" --output json)

    for policy_name in $(echo "$inline_policies" | jq -r '.[]'); do
        echo "  Deleting inline policy: $policy_name"
        aws iam delete-role-policy --role-name "$role_name" --policy-name "$policy_name"
    done

    instance_profiles=$(aws iam list-instance-profiles-for-role --role-name "$role_name" --query "InstanceProfiles[*].{Name:InstanceProfileName}" --output json)

    for profile in $(echo "$instance_profiles" | jq -c '.[]'); do
        profile_name=$(echo "$profile" | jq -r '.Name')
        echo "  Removing role from instance profile: $profile_name"
        aws iam remove-role-from-instance-profile --instance-profile-name "$profile_name" --role-name "$role_name"

        echo "  Deleting instance profile: $profile_name"
        aws iam delete-instance-profile --instance-profile-name "$profile_name"
    done

    echo "  Deleting role: $role_name"
    aws iam delete-role --role-name "$role_name"

    echo "✓ Removed Role: $role_name"
done

echo "All 'nodo-mono' IAM Roles have been removed"

#
##
###
####
##### Security Groups
echo ""
echo "Handling security group operations..."

# Delete all security groups starting with 'nodo-mono-sg'
echo ""
echo "Looking for security groups with name starting with 'nodo-mono-sg'..."

# Find all security groups with names starting with 'nodo-mono-sg'
sg_list=$(aws ec2 describe-security-groups --filters "Name=group-name,Values=nodo-mono-sg*" --query "SecurityGroups[*].{ID:GroupId,Name:GroupName}" --output json)

echo "Found the following security groups to remove:"
echo "$sg_list" | jq -r '.[].Name'
echo "---"

# Delete each security group
echo "$sg_list" | jq -c '.[]' | while read -r sg; do
    sg_id=$(echo "$sg" | jq -r '.ID')
    sg_name=$(echo "$sg" | jq -r '.Name')

    echo "Deleting security group: $sg_name ($sg_id)"
    aws ec2 delete-security-group --group-id "$sg_id"
    echo "✓ Removed security group: $sg_name ($sg_id)"
done

echo "All 'nodo-mono-sg' security groups have been removed"

# Step 3: Delete security groups with names starting with 'nodo-mono-alb'
sleep 20
echo ""
echo "Looking for security groups with name starting with 'nodo-mono-alb'..."

# Find all security groups with names starting with 'nodo-mono-alb'
alb_sg=$(aws ec2 describe-security-groups --filters "Name=group-name,Values=nodo-mono-alb*" --query "SecurityGroups[*].{ID:GroupId,Name:GroupName}" --output json)

echo "Found the following security groups to remove:"
echo "$alb_sg" | jq -r '.[].Name'
echo "---"

# Delete each security group
echo "$alb_sg" | jq -c '.[]' | while read -r sg; do
    sg_id=$(echo "$sg" | jq -r '.ID')
    sg_name=$(echo "$sg" | jq -r '.Name')

    echo "Deleting security group: $sg_name ($sg_id)"
    aws ec2 delete-security-group --group-id "$sg_id"
    echo "✓ Removed security group: $sg_name ($sg_id)"
done

echo "All 'nodo-mono-alb' security groups have been removed"

echo ""

#
##
###
####
##### Target Groups
target_groups=$(aws elbv2 describe-target-groups --query "TargetGroups[?starts_with(TargetGroupName, 'nodo-mono')].{ARN:TargetGroupArn,Name:TargetGroupName}" --output json)

echo "Found the following Target Groups to remove:"
echo "$target_groups" | jq -r '.[].Name'
echo "---"

echo "$target_groups" | jq -c '.[]' | while read -r tg; do
    tg_arn=$(echo "$tg" | jq -r '.ARN')
    tg_name=$(echo "$tg" | jq -r '.Name')

    echo "Processing Target Group: $tg_name"

    echo "  Deleting Target Group..."
    aws elbv2 delete-target-group --target-group-arn "$tg_arn"

    echo "✓ Removed Target Group: $tg_name"
done

echo "All 'nodo-mono' Target Groups have been removed"

#
##
###
####
##### Second Sec Groups
echo ""
echo "Second sec groups search -> Looking for security groups with name starting with 'nodo-mono-alb'..."

# Find all security groups with names starting with 'nodo-mono-alb'
alb_sg=$(aws ec2 describe-security-groups --filters "Name=group-name,Values=nodo-mono-alb*" --query "SecurityGroups[*].{ID:GroupId,Name:GroupName}" --output json)

echo "Found the following security groups to remove:"
echo "$alb_sg" | jq -r '.[].Name'
echo "---"

# Delete each security group
echo "$alb_sg" | jq -c '.[]' | while read -r sg; do
    sg_id=$(echo "$sg" | jq -r '.ID')
    sg_name=$(echo "$sg" | jq -r '.Name')

    echo "Deleting security group: $sg_name ($sg_id)"
    aws ec2 delete-security-group --group-id "$sg_id"
    echo "✓ Removed security group: $sg_name ($sg_id)"
done

echo "All 'nodo-mono-alb' security groups have been removed"
