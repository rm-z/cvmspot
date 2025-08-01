#!/bin/bash

upload_path="/root"

mv ${upload_path}/frp/frps /usr/local/bin/frps
chmod +x /usr/local/bin/frps
mkdir -p /etc/frp && mv ${upload_path}/frp/frps.toml /etc/frp/frps.toml

cat > /etc/systemd/system/frps.service <<EOF
[Unit]
Description=Frp Server Service
After=network-online.target

[Service]
Type=simple
Restart=on-failure
RestartSec=5s
ExecStart=/usr/local/bin/frps -c /etc/frp/frps.toml
ExecReload=/usr/local/bin/frps reload -c /etc/frp/frps.toml
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now frps
systemctl status frps
