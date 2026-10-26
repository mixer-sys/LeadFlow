### For fast local start
```
cd LeadFlow/deploy_local/
sudo docker compose --profile app --profile observability up

if [[ -f .env ]]; then
  set -a
  source .env
  set +a
fi

go run ./cmd/api..


if [[ -f .env ]]; then
  set -a
  source .env
  set +a
fi

go run ./cmd/worker..

./test.sh



```