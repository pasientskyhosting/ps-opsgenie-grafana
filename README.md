# gropsgenie
OpsGenie alert list in grafana using the grafana-simple-json-datasource

You need these environment variables:
- `OPSGENIE_API_KEY` - your OpsGenie API key
- `OPSGENIE_FETCH_INTERVAL` - how often (in seconds) to fetch alerts from OpsGenie (default: 60)
- `METRICS_PORT` - the port gropsgenie should listen on (default: 3001)

## Usage
```
docker run -d -p 3001:3001 -e OPSGENIE_API_KEY=00000000-0000-0000-0000-000000000000 pasientskyhosting/ps-opsgenie-grafana:1.0
```
