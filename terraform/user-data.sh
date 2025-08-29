#!/bin/bash

# User data script for nex-gen-cms deployment
# This script runs on every boot and is idempotent

set -euo pipefail

LOG_FILE="/var/log/nexgencms-setup.log"
APP_DIR="/opt/nex-gen-cms"
APP_USER="app"
ENV_FILE="/etc/nexgencms.env"
SERVICE_NAME="nexgencms"

# Logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE"
}

log "Starting nex-gen-cms setup script"

# Update system and install required packages
log "Installing required packages"
dnf update -y
dnf install -y git nginx golang certbot python3-certbot-nginx firewalld

# Create app user if not exists
if ! id "$APP_USER" &>/dev/null; then
    log "Creating app user: $APP_USER"
    useradd --system --shell /bin/false --home-dir "$APP_DIR" --create-home "$APP_USER"
fi

# Create necessary directories
log "Creating application directories"
mkdir -p "$APP_DIR" "/var/log/nexgencms"
chown "$APP_USER:$APP_USER" "$APP_DIR" "/var/log/nexgencms"

# Write environment files
log "Writing environment configuration"
cat > "$ENV_FILE" << 'EOF'
DB_SERVICE_ENDPOINT=${db_service_endpoint}
DB_SERVICE_TOKEN=${db_service_token}
EOF
chown "$APP_USER:$APP_USER" "$ENV_FILE"
chmod 600 "$ENV_FILE"

# Clone or update repository
log "Setting up application code"
cd "$APP_DIR"
if [ -d .git ]; then
    log "Updating existing repository"
    sudo -u "$APP_USER" git fetch origin
    sudo -u "$APP_USER" git reset --hard origin/${repo_branch}
else
    log "Cloning repository"
    # Remove directory contents if not empty
    rm -rf "$APP_DIR"/*
    rm -rf "$APP_DIR"/.*
    sudo -u "$APP_USER" git clone -b ${repo_branch} ${repo_url} .
fi

# Create .env file in application directory for godotenv (after git clone)
log "Creating .env file for application"
cat > "$APP_DIR/.env" << 'EOF'
DB_SERVICE_ENDPOINT=${db_service_endpoint}
DB_SERVICE_TOKEN=${db_service_token}
EOF
chown "$APP_USER:$APP_USER" "$APP_DIR/.env"
chmod 600 "$APP_DIR/.env"

# Build the application
log "Building application"
sudo -u "$APP_USER" go mod download
sudo -u "$APP_USER" go build -o "$APP_DIR/nex-gen-cms" ./cmd

# Create systemd service file
log "Creating systemd service"
cat > "/etc/systemd/system/$SERVICE_NAME.service" << 'EOF'
[Unit]
Description=Nex Gen CMS Application
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=app
Group=app
WorkingDirectory=/opt/nex-gen-cms
EnvironmentFile=/etc/nexgencms.env
ExecStart=/opt/nex-gen-cms/nex-gen-cms
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and start service
log "Starting application service"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

# Configure nginx - check if SSL certificates exist
log "Configuring nginx"

if [ -f "/etc/letsencrypt/live/${domain}/fullchain.pem" ]; then
    log "SSL certificates found, configuring HTTPS"
    cat > /etc/nginx/conf.d/nexgencms.conf << EOF
server {
    listen 80;
    server_name ${domain};

    # Allow Let's Encrypt challenges
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # Redirect all other HTTP traffic to HTTPS
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl;
    http2 on;
    server_name ${domain};

    # SSL certificate paths (managed by certbot)
    ssl_certificate /etc/letsencrypt/live/${domain}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${domain}/privkey.pem;

    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-SHA256:ECDHE-RSA-AES256-SHA384;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;

    # Gzip compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;

    # Proxy to the Go application
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_http_version 1.1;
        proxy_intercept_errors on;
    }
}
EOF
else
    log "No SSL certificates found, configuring HTTP-only for now"
    cat > /etc/nginx/conf.d/nexgencms.conf << EOF
server {
    listen 80;
    server_name ${domain};

    # Allow Let's Encrypt challenges
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # Proxy to the Go application (HTTP-only until SSL is set up)
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_http_version 1.1;
        proxy_intercept_errors on;
    }
}
EOF
fi

# Test nginx configuration
if nginx -t; then
    log "Nginx configuration is valid"
    systemctl enable nginx
    systemctl restart nginx
else
    log "ERROR: Nginx configuration is invalid"
    exit 1
fi

# Configure firewall if enabled
if systemctl is-active firewalld >/dev/null 2>&1; then
    log "Configuring firewall"
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
    firewall-cmd --permanent --add-service=ssh
    firewall-cmd --reload
fi

# Setup SSL certificate with certbot
log "Setting up SSL certificate"
if [ ! -f "/etc/letsencrypt/live/${domain}/fullchain.pem" ]; then
    log "Obtaining SSL certificate for ${domain}"
    certbot --nginx -d ${domain} \
        --non-interactive \
        --agree-tos \
        --email ${letsencrypt_email} \
        --no-redirect
    
    if [ $? -eq 0 ]; then
        log "SSL certificate obtained successfully, reconfiguring nginx for HTTPS"
        # Reconfigure nginx with SSL after obtaining certificates
        cat > /etc/nginx/conf.d/nexgencms.conf << EOF
server {
    listen 80;
    server_name ${domain};

    # Allow Let's Encrypt challenges
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # Redirect all other HTTP traffic to HTTPS
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl;
    http2 on;
    server_name ${domain};

    # SSL certificate paths (managed by certbot)
    ssl_certificate /etc/letsencrypt/live/${domain}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${domain}/privkey.pem;

    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-SHA256:ECDHE-RSA-AES256-SHA384;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;

    # Gzip compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;

    # Proxy to the Go application
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_http_version 1.1;
        proxy_intercept_errors on;
    }
}
EOF
        nginx -t && systemctl reload nginx
        log "HTTPS setup completed successfully"
    else
        log "ERROR: Failed to obtain SSL certificate, continuing with HTTP-only"
        # Continue without SSL for now
    fi
else
    log "SSL certificate already exists"
fi

# Ensure certbot renewal timer is enabled
systemctl enable certbot-renew.timer
systemctl start certbot-renew.timer

# Create a renewal hook to reload nginx
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh << 'EOF'
#!/bin/bash
systemctl reload nginx
EOF
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh

log "Setup completed successfully"
log "Application should be available at: https://${domain}"
