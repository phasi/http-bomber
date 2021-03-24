# http-bomber
Make HTTP requests to one or multiple endpoints and send results to Elasticsearch

## Prerequisites

You need to install the following software:

- bash & git (for building)
- Go (for building)
- Docker
    - Local elasticsearch + kibana started ([with this script](resources/elasticsearch-kibana.sh))

```bash
# Start elastic+kibana
resources/elasticsearch-kibana.sh deploy
# To destroy
resources/elasticsearch-kibana.sh destroy
```

### Compiling/Building

You need Go language installed on your computer before starting.

You can either build manually with go or use the "wrapper" build script:

```bash
./build-tool.sh build
```

## Example run


```bash

# Go to dist folder created by the build script
cd dist

# Example command
./http-bomber \
-url http://domainone.com,http://domaintwo.com \
-duration 10 \
-timeout 3 \
-export \
-el-url http://localhost:9200/indexname/_doc \
-debug

```
