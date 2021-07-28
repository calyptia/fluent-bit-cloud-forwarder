## Forwarder

Forwards metrics from Fluent Bit to Calyptia Cloud.

## Releases

[Check the releases page](https://github.com/calyptia/fluent-bit-cloud-forwarder/releases).

## Build instrutions

```
sudo dpkg -i external/*.deb
```
```
go build ./cmd/forwarder
```

## Usage

```
./forwarder -h
```

```
Forwards metrics from Fluent Bit agent to Calyptia Cloud.
It stores some persisted data about Cloud registration at "data" directory.
Flags:
  -agent-config-file string
        Fluentbit agent config file (default "fluent-bit.conf")
  -agent-hostname string
        Agent hostname. If empty, a random one will be generated
  -agent-machine-id string
        Agent host machine ID. If empty, a random one will be generated
  -agent-pull-interval duration
        Interval to pull Fluent Bit agent and forward metrics to Cloud (default 5s)
  -agent-url string
        Fluent Bit agent URL (default "http://localhost:2020")
  -cloud-url string
        Calyptia Cloud API URL (default "https://cloud-api-dev.calyptia.com/")
  -project-token string
        Project token from Calyptia Cloud fetched from "POST /api/v1/tokens" or from "GET /api/v1/tokens?last=1"
```

## Docker

To run it with Docker, first go to https://config-viewer-ui-dev.herokuapp.com and create a new project token.

Copy `.env.example` into a new file `.env`.
```
cp .env.example .env
```

Edit this `.env` file and add your Calyptia Cloud project token.

Then just:
```
docker-compose up
```
