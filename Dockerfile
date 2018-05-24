FROM golang:alpine AS build-env
RUN apk --update add ca-certificates
WORKDIR /go/src/github.com/TechemyLtd/ecs-instance-drainer
ADD . .
RUN CGO_ENABLED=0 GOOS=linux
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -o ecs-instance-drainer .

FROM scratch
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build-env /go/src/github.com/TechemyLtd/ecs-instance-drainer/ecs-instance-drainer /ecs-instance-drainer
ENTRYPOINT ["/ecs-instance-drainer"]
