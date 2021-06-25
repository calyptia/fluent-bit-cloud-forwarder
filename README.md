## Forwarder

Forwards metrics from Fluent Bit to Calyptia Cloud.

## Build instrutions

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
  -api-key string
        Calyptia Cloud API key
  -cloud string
        Calyptia Cloud API URL (default "http://localhost:8080")
  -hostname string
        Agent hostname. If empty, a random one will be generated
  -interval duration
        Interval to pull Fluent Bit agent and forward metrics to Cloud (default 5s)
```
