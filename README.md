# jackproxy

A custom hijacking reverse-proxy for Percy's internal rendering environment.

## Development

Install dependencies:

```bash
$ glide install
```

```bash
$ jackproxy --port 8080 --proxymap testdata/proxymap.json
$ curl -x localhost:8080 http://proxyme.local/
```