#!/usr/bin/env bash
# End-to-end deploy: terraform apply -> build web -> sync to S3 -> invalidate
# CloudFront -> redeploy API on the EC2 host via SSM -> smoke-check /health.
#
#   AWS_PROFILE=shardx ./deploy/deploy.sh            # everything
#   AWS_PROFILE=shardx ./deploy/deploy.sh web        # web only
#   AWS_PROFILE=shardx ./deploy/deploy.sh api        # api/worker only (git pull + rebuild on host)
set -euo pipefail
cd "$(dirname "$0")/.."
TF="terraform -chdir=infrastructure/terraform"
what="${1:-all}"

if [[ "$what" == "all" ]]; then
  $TF apply -input=false
fi

APP_URL=$($TF output -raw app_url)
BUCKET=$($TF output -raw web_bucket_name)
DIST_ID=$($TF output -raw cloudfront_distribution_id)
INSTANCE=$($TF output -raw host_instance_id)
REGION=$($TF output -raw aws_region 2>/dev/null || echo us-east-1)

if [[ "$what" == "all" || "$what" == "web" ]]; then
  echo "== web -> s3://$BUCKET"
  (cd apps/web && npm ci --no-audit --no-fund && VITE_API_BASE_URL= npm run build)
  aws s3 sync apps/web/dist "s3://$BUCKET" --delete --cache-control "public,max-age=31536000,immutable" --exclude index.html
  aws s3 cp apps/web/dist/index.html "s3://$BUCKET/index.html" --cache-control "no-cache"
  aws cloudfront create-invalidation --distribution-id "$DIST_ID" --paths "/index.html" >/dev/null
fi

if [[ "$what" == "all" || "$what" == "api" ]]; then
  echo "== api -> $INSTANCE (waiting for SSM agent)"
  for _ in $(seq 1 60); do
    aws ssm describe-instance-information --region "$REGION" \
      --filters "Key=InstanceIds,Values=$INSTANCE" --query 'InstanceInformationList[0].PingStatus' --output text 2>/dev/null | grep -q Online && break
    sleep 10
  done
  CMD_ID=$(aws ssm send-command --region "$REGION" --instance-ids "$INSTANCE" \
    --document-name AWS-RunShellScript --comment "shardx redeploy" \
    --parameters 'commands=["/usr/local/bin/shardx-redeploy"],executionTimeout=["1800"]' \
    --query Command.CommandId --output text)
  aws ssm wait command-executed --region "$REGION" --command-id "$CMD_ID" --instance-id "$INSTANCE" || true
  aws ssm get-command-invocation --region "$REGION" --command-id "$CMD_ID" --instance-id "$INSTANCE" \
    --query '[Status,StandardErrorContent]' --output text | head -20
fi

echo "== smoke"
for _ in $(seq 1 30); do
  curl -sf "$APP_URL/health" && { echo; echo "OK: $APP_URL"; exit 0; }
  sleep 10
done
echo "health check did not pass at $APP_URL" >&2; exit 1
