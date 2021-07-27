FROM golang:stretch as build

ARG GITHUB_PERSONAL_ACCESS_TOKEN_NAME
ARG GITHUB_PERSONAL_ACCESS_TOKEN

ENV GITHUB_PERSONAL_ACCESS_TOKEN_NAME=${GITHUB_PERSONAL_ACCESS_TOKEN_NAME}
ENV GITHUB_PERSONAL_ACCESS_TOKEN=${GITHUB_PERSONAL_ACCESS_TOKEN}

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
RUN go env -w GOPRIVATE=github.com/calyptia/cloud
RUN git config --global url."https://${GITHUB_PERSONAL_ACCESS_TOKEN_NAME}:${GITHUB_PERSONAL_ACCESS_TOKEN}@github.com".insteadOf "https://github.com"
RUN go mod download
RUN go build -ldflags "-linkmode external -extldflags -static" -o /forwarder ./cmd/forwarder

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /forwarder /forwarder

ENTRYPOINT [ "/forwarder" ]
