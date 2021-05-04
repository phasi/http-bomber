# HTTP Bomber Usage Documentation

This documentation focuses on how to use HTTP Bomber. 

## Download package

You can get a binary package from [releases page](https://github.com/phasi/http-bomber/releases) to get started.

### Download via CLI (linux and mac)

```bash
# Linux
wget https://github.com/phasi/http-bomber/releases/download/2.1.0/http-bomber_linux_amd64 && mv http-bomber_linux_amd64 http-bomber

# Mac
wget https://github.com/phasi/http-bomber/releases/download/2.1.0/http-bomber_darwin_amd64 && mv http-bomber_linux_amd64 http-bomber
```

## Permissions

You might need to give http-bomber binary the necessary permissions to run. On linux/mac you can do this by running:

```bash
chmod +x http-bomber
```

## Simple usage

Below example will make a 10 second test to example.org and (-debug) will print each request status and round trip time on the screen.

```bash
./http-bomber -url http://example.org -duration 10 -debug
```

## Options

The complete list of options for HTTP Bomber.

### URL

The URL to target. You can pass multiple URLs separated by a comma

```bash
-url <url1,url2>
```

### Network stack

Network stack to be used. Default is TCP/IPv4

```bash
-n <tcp4|tcp6>
```

### Headers

You can add custom headers to the requests. By default X-Tested-With and User-Agent headers are added.

You can add multiple headers separated by a comma.

```bash
# one header
-headers "<Header>:<value>"

#multiple headers
-headers "<Header>:<value>,<Header>:<value>"
```

## Timeout

Timeout in seconds for a single request.

```bash
-timeout <int>
```

## Duration

HTTP test duration in seconds.

```bash
-duration <int>
```

## Interval

HTTP request interval in milliseconds. Tip: Don't set this too small or you might be blocked by a firewall. Default is 1000 (one second).

```bash
-interval <int>
```

## TLS verification

TLS certificates are verified by default. If you want to disable the verification add the following option:

```bash
-tls-skip-verify
```

## Follow HTTP redirects

By default HTTP Bomber does not follow HTTP redirects. If you want to enable this, add the following flag:

```bash
-follow-redirects
```

## Always attempt HTTP2

This force attempts HTTP2 in all scenarios. Read more [in golang's documentation](https://golang.org/src/net/http/transport.go?s=3377:11444#L84)

Enable by setting this option:

```bash
-force-try-http2
```

## Debug logging

Enable debug logging (including logging of every single request to stdout)

```bash
-debug
```


## MODULE: Elasticsearch

For exporting the test results to Elasticsearch you need to define one or both of the following options:

### Export against Elasticsearch bulk API

This will configure exporting directly against elasticsearch bulk API.

```bash
-elastic-export
```

You also need to define url. Otherwise the default will be used (http://localhost:9200)

```bash
-elastic-url <url>
```

### Export into a file (in elasticsearch bulk API format)

This will configure exporting to a file. The elasticsearch bulk API format will be used so that you may later export the file manually with a tool of your choice.

```bash
-elastic-export-to-file
```

You also need to define the filepath:

```bash
-elastic-export-filepath <path/to/file.json>
```

### Defining index

In elasticsearch data is structured in indexes/indices. You need to define under which index your data will be added.
Tip: It might be reasonable to gather your test results under separate indexes to avoid confusion.

```bash
-elastic-index <string>
```

## MODULE: IP Stack

For tracing IP address geolocation and other details you may create a free API key at [https://ipstack.com/](https://ipstack.com/). This module will use your API key to fetch location data for each unique IP address. When running a long test your domain might get resolved to multiple IP addresses during the test.

### Enabling IP Stack module

```bash
-ipstack
```

### Configure IP Stack API key

```bash
-ipstack-apikey <string>
```

### Configure timeout for IP Stack

Timeout in seconds.

```bash
-ipstack-timeout <int>
```