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
It stores some persisted data about Cloud registration at "diskv-data" directory.
Flags:
  -agent string
        Fluent Bit agent URL (default "http://localhost:2020")
  -agent-config string
        Fluentbit agent config file
  -cloud string
        Calyptia Cloud API URL (default "http://localhost:5000")
  -hostname string
        Agent hostname. If empty, a random one will be generated
  -interval duration
        Interval to pull Fluent Bit agent and forward metrics to Cloud (default 5s)
  -machine-id string
        Agent host machine ID. If empty, a random one will be generated
  -project-token string
        Project token from Calyptia Cloud fetched from "POST /api/v1/tokens" or from "GET /api/v1/tokens?last=1"
```
