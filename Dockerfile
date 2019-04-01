FROM golang:alpine AS build-env
RUN apk --update add ca-certificates

FROM scratch
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD ecs-instance-drainer /ecs-instance-drainer
ENTRYPOINT ["/ecs-instance-drainer"]
