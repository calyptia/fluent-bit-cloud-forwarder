FROM golang:stretch as build

ENV PROJECT_TOKEN=""
ENV CLOUD_URL=https://cloud-api-dev.calyptia.com/
ENV AGENT_URL=http://fluentbit:2020
ENV AGENT_PULL_INTERVAL=5s
ENV AGENT_CONFIG_FILE=fluent-bit.conf
ENV AGENT_HOSTNAME=""
ENV AGENT_MACHINE_ID=""

WORKDIR /go/src/github.com/calyptia/fluent-bit-cloud-forwarder
COPY . .
RUN apt update -yyq && apt -yyq install ca-certificates
RUN update-ca-certificates
RUN dpkg -i external/*.deb
RUN go mod download
RUN go build -ldflags "-linkmode external -extldflags -static" -o /forwarder ./cmd/forwarder

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /forwarder /fluent-bit-cloud-forwarder

ENTRYPOINT [ "/fluent-bit-cloud-forwarder" ]
