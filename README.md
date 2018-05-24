# ecs-instance-drainer
Deployed to ECS clusters to handle ASG notifications

## Build Source
### Windows
```bash
$env:GOOS = "linux"; $env:GOARCH = "amd64"; go build
```

### Linux
```bash
GOOS=target-OS GOARCH=amd64 go build
docker build -t "ecs-instance-drainer" .
```

## Run Tests
```bash
go test -cover ./...
```

## Build Container
```bash
docker build -t "ecs-instance-drainer" .
```

## Run Container

* Replace ${QUEUE} with the URL of the Lifecycle Queue
* Replace ${CLUSTER} with the name of the ECS cluster
```bash
docker run --memory-reservation="20m" -e "LIFECYCLE_QUEUE=${QUEUE}" -e "CLUSTER=${CLUSTER}" -e "DRAINER_TIMEOUT=1h" ecs-instance-drainer--memory-reservation
```
