# http-bomber
Make HTTP requests to one or multiple endpoints and send results to Elasticsearch

## Usage documentation
See ["USAGE.md"](./USAGE.md)

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
./build-tool.sh build <linux|darwin|windows>
```

## Example run


```bash

# Go to dist folder created by the build script
cd dist

# Example command
./http-bomber \
-url http://example.org,https://google.com \
-n tcp4 \
-headers "CustomHeader:IamAValue,X-Something:another_value" \
-timeout 2 \
-duration 30 \
-interval 1000 \
-elastic-export \
-elastic-index testidata \
-elastic-url http://localhost:9200 \
-elastic-export-to-file \
-elastic-export-filepath ~/results.json \
-ipstack \
-ipstack-apikey "yourapikey" \
-ipstack-timeout 5 \
-tls-skip-verify \
-follow-redirects \
-force-try-http2 \
-debug



```

## Disclaimer

This is actually my first go-project... And its not actively developed.. But it does what it promises :-)
