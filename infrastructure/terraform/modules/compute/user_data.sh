#!/bin/bash
# Bootstraps the ShardX host: docker + compose, clone repo, wait for the env
# parameter (written after CloudFront exists), start the stack.
set -euxo pipefail
exec > >(tee /var/log/shardx-bootstrap.log) 2>&1

# 2 GB swap: `docker compose build` of the Go API needs more than a t3.small's RAM.
if [ ! -f /swapfile ]; then
  fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
  echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

dnf install -y docker git
systemctl enable --now docker
# compose + buildx CLI plugins (AL2023 ships neither; compose build needs buildx).
mkdir -p /usr/local/lib/docker/cli-plugins
curl -fsSL "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
curl -fsSL "$(curl -fsSL https://api.github.com/repos/docker/buildx/releases/latest | grep -o 'https://[^"]*linux-amd64' | head -1)" \
  -o /usr/local/lib/docker/cli-plugins/docker-buildx
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose /usr/local/lib/docker/cli-plugins/docker-buildx

install -d /opt/shardx
if [ ! -d /opt/shardx/.git ]; then
  git clone --depth 1 --branch "${repo_ref}" "${repo_url}" /opt/shardx
fi

# Redeploy helper used by deploy/deploy.sh via SSM Run Command.
cat > /usr/local/bin/shardx-redeploy <<'SCRIPT'
#!/bin/bash
set -euo pipefail
cd /opt/shardx
git fetch --depth 1 origin "${repo_ref}" && git reset --hard FETCH_HEAD
aws ssm get-parameter --region "${region}" --name "${param_name}" --with-decryption \
  --query Parameter.Value --output text > deploy/.env
chmod 600 deploy/.env
docker compose -f deploy/docker-compose.yml up -d --build --remove-orphans
docker image prune -f
SCRIPT
chmod +x /usr/local/bin/shardx-redeploy

# The env parameter is created after the CloudFront distribution; poll for it.
for i in $(seq 1 90); do
  if aws ssm get-parameter --region "${region}" --name "${param_name}" --with-decryption >/dev/null 2>&1; then
    break
  fi
  sleep 10
done

/usr/local/bin/shardx-redeploy
