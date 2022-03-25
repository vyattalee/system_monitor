curl -X PUT -d '{ "id": "pushgateway_mem", "name": "pushgateway_memory", "address": "localhost", "port": 9091, "tags": ["memory_usage"], "checks": [{"http": "http://localhost:9091/metrics", "interval": "5s"}]}' http://localhost:8500/v1/agent/service/register

