#!/bin/bash
set -e

exec > >(tee /var/log/user-data.log) 2>&1 # for debugging startup -> cat /var/log/user-data.log <- (only on public subnets)
export HOME=/root
APP_DIR="$HOME/nodofinance"

# cloudWatch agent
LOG_GROUP_NAME=$(aws logs describe-log-groups --region eu-central-1 --log-group-name-prefix "/ec2/nodo-mono" --query "logGroups[0].logGroupName" --output text)

echo "Installing CloudWatch agent..."
sudo yum install -y amazon-cloudwatch-agent

sudo touch /var/log/nodofinance.log
sudo chmod 644 /var/log/nodofinance.log

sudo bash -c "cat > /opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json << EOF
{
  \"agent\": {
    \"metrics_collection_interval\": 300,
    \"run_as_user\": \"root\"
  },
  \"metrics\": {
    \"metrics_collected\": {
      \"disk\": {
        \"measurement\": [
          \"used_percent\"
        ],
        \"resources\": [
          \"/\"
        ],
        \"ignore_file_system_types\": [
          \"sysfs\", \"devtmpfs\"
        ]
      },
      \"mem\": {
        \"measurement\": [
          \"mem_used_percent\"
        ]
      }
    },
    \"append_dimensions\": {
      \"InstanceId\": \"\\\${aws:InstanceId}\"
    }
  },
  \"logs\": {
    \"logs_collected\": {
      \"files\": {
        \"collect_list\": [
          {
            \"file_path\": \"/var/log/user-data.log\",
            \"log_group_name\": \"$LOG_GROUP_NAME\",
            \"log_stream_name\": \"user-data-{instance_id}\"
          },
          {
            \"file_path\": \"/var/log/nodofinance.log\",
            \"log_group_name\": \"$LOG_GROUP_NAME\",
            \"log_stream_name\": \"application-{instance_id}\"
          }
        ]
      }
    }
  }
}
EOF"

sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a fetch-config -m ec2 -s -c file:/opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json

# get compiled
mkdir -p $APP_DIR
echo "Downloading compiled files from S3..."
aws s3 cp s3://compiled-prod-nodofinance/build.tar.gz $APP_DIR/build.tar.gz
echo "Extracting files..."
tar -xzvf $APP_DIR/build.tar.gz -C $APP_DIR
echo "Cleaning up..."
rm $APP_DIR/build.tar.gz

# system optimizations for network performance
echo "net.core.somaxconn=65535" >>/etc/sysctl.conf
echo "net.ipv4.ip_local_port_range=1024 65535" >>/etc/sysctl.conf
echo "net.ipv4.tcp_tw_reuse=1" >>/etc/sysctl.conf
echo "kernel.sched_autogroup_enabled=0" >>/etc/sysctl.conf
echo "net.core.netdev_max_backlog=65536" >>/etc/sysctl.conf
sysctl -p

# set file limits for high connection counts
echo "* soft nofile 65535" >>/etc/security/limits.conf
echo "* hard nofile 65535" >>/etc/security/limits.conf

# create systemd service for auto-start on reboot
cat >/etc/systemd/system/nodofinance.service <<EOF
[Unit]
Description=nodo.finance server
After=network.target

[Service]
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/nodofinance
Restart=always
RestartSec=5
LimitNOFILE=65535
Environment="HOME=$APP_DIR"
StandardOutput=append:/var/log/nodofinance.log
StandardError=append:/var/log/nodofinance.log

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/logrotate.d/nodofinance <<EOF
/var/log/nodofinance.log {
  daily
  rotate 2
  compress
  size 50M
  missingok
  notifempty
  create 0644 root root
  maxage 1
}
EOF

# load ssm .env
>$APP_DIR/.env

PARAMS=(
    "VITE_BASE_URL"
    "VITE_BASE_URL_DEV"
    "VITE_STRIPE_PK"
    "VITE_STRIPE_PK_TEST"
    "VITE_MAX_PERIODS_PER_TICKER"
    "VITE_MAX_TICKERS"
    "VITE_EMAIL_CONTACT"
)

for param in "${PARAMS[@]}"; do
    param_name=$(basename "$param")
    param_value=$(aws ssm get-parameter --name "$param" --with-decryption --query "Parameter.Value" --output text)
    echo "$param_name=$param_value" >>$APP_DIR/.env
done

echo "DEV_MODE=false" >>"$APP_DIR/.env"
echo "AWS_REGION=eu-central-1" >>"$APP_DIR/.env"

# enable and start the service
systemctl daemon-reload
systemctl enable nodofinance.service
systemctl start nodofinance.service
systemctl status nodofinance.service
