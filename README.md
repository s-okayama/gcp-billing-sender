# GCP Billing
## Set Environment
```bash
export BIGQUERY_DATASET=gcp_billing_export_v1_XXXXXX_XXXXX_XXXXX
export GCP_PROJECT_ID=xxxxxxxxxxx
export DATADOG_API_KEY=xxxxxxxxxxxxxx
```

## Run
```bash
go run main.go
go run -o main main.go
```
## Build
```bash
docker buildx build -t gcp_billing:v1.1.1 . \
--platform "linux/amd64"
 
## Deploy
```bash
# diff
helm template ./ | kubectl diff -f -
# apply
helm template ./ | kubectl apply -f -
```
## Debug
```bash
kubectl run okayama-test -n sre --tty -i --image=gcp_billing:v1.1.1 /bin/bash

```