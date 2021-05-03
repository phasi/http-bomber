package main

// Imports
import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"http-bomber/elasticsearch"
	"http-bomber/httptest"
	"http-bomber/ipstack"
	"http-bomber/logging"
)

// GLOBALS
// logger
var logger logging.Logger

// debug mode
var debug bool

// log file
var logFilePath string = "./httptest.log"

// channel for test results
var exportedDataChan chan []*httptest.Result

// wait group for goroutines
var wg sync.WaitGroup

// Flags
var showVersion bool = false
var url string
var hdrs string
var headers http.Header = make(http.Header)
var duration int
var timeout int
var interval int
var networkStack string = "tcp"
var tlsVerify bool = false
var followRedirects bool = false
var forceAttemptHTTP2 bool = false

// MODULE GLOBALS
// Elasticsearch
var elConfig elasticsearch.Config

// IP Stack
var ipstackConfig ipstack.Settings

// Configure application logging
func configLogging() {
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("Could not open logfile")
		logger.Init(os.Stdout)
	} else {
		logger.Init(os.Stdout, logFile)
	}
}

// Parse headers from string to http.Header map
func parseHeadersFlag(headers *string, parsedHeaders *http.Header) {
	hdrsSlice := strings.Split(*headers, ",")
	for _, v := range hdrsSlice {
		hdr := strings.Split(v, ":")
		parsedHeaders.Add(hdr[0], hdr[1])
	}
}

// Initial operations
func init() {
	// configure logging
	configLogging()

	// READ FLAGS
	// Version info
	flag.BoolVar(&showVersion, "version", false, "Show version info")

	// Common flags
	flag.BoolVar(&debug, "debug", false, "This flag turns debugging on.")
	flag.StringVar(&networkStack, "n", "tcp4", "Network stack")
	flag.StringVar(&url, "url", "http://localhost", "URL to test. Add multiple URLs separated by a comma (no whitespaces in between)")
	flag.StringVar(&hdrs, "headers", fmt.Sprintf("X-Tested-With:http-bomber/%s", AppVersion), "Additional headers example-> Host:localhost,X-Custom-Header:helloworld")
	flag.IntVar(&duration, "duration", 10, "Test duration in seconds")
	flag.IntVar(&timeout, "timeout", 5, "Connection timeout in seconds")
	flag.IntVar(&interval, "interval", 1000, "Request interval in milliseconds")
	flag.BoolVar(&tlsVerify, "tls-skip-verify", false, "Skip TLS certificate validation.")
	flag.BoolVar(&followRedirects, "follow-redirects", false, "Follow HTTP Redirects.")
	flag.BoolVar(&forceAttemptHTTP2, "force-try-http2", false, "Force attempt HTTP2.")

	// MODULE FLAGS
	// Elasticsearch
	flag.StringVar(&elConfig.URL, "elastic-url", "http://localhost:9200", "Elastic search URL")
	flag.StringVar(&elConfig.IndexName, "elastic-index", "testdata", "Elasticsearch index name")
	flag.BoolVar(&elConfig.Export, "elastic-export", false, "Export data to elasticsearch")
	flag.BoolVar(&elConfig.ExportToFile, "elastic-export-to-file", false, "Export data to file in elasticsearch format")
	flag.StringVar(&elConfig.ExportFilePath, "elastic-export-filepath", "/tmp/http-bomber-results.json", "Specify filepath for Elasticsearch export")
	// IPStack
	flag.BoolVar(&ipstackConfig.UseIPStack, "ipstack", false, "Use IPStack for example for getting geolocation details")
	flag.StringVar(&ipstackConfig.APIKey, "ipstack-apikey", "1234", "Your personal IPStack API key")
	flag.IntVar(&ipstackConfig.Timeout, "ipstack-timeout", 3, "IPStack connect timeout")

	// Parse flags
	flag.Parse()

	// Print version info and exit
	if showVersion {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	// Parse extra headers
	headers.Add("User-Agent", fmt.Sprintf("http-bomber/%s", AppVersion))
	parseHeadersFlag(&hdrs, &headers)

}

func main() {

	// Log program start
	logger.Info(fmt.Sprint("Starting HTTP Bomber ", AppVersion))

	// Get URLs
	urls := strings.Split(url, ",")
	// Make channel for results
	exportedDataChan = make(chan []*httptest.Result, len(urls))
	// Set the number of wait groups based on the quantity of URLs provided by the user
	wg.Add(len(urls))

	// Goroutines for each url provided
	for i := 0; i < len(urls); i++ {
		logger.Info(fmt.Sprintf("Starting test %v (URL: %s)", i+1, urls[i]))
		settings := httptest.Settings{
			URL:               urls[i],
			Duration:          time.Duration(duration),
			Timeout:           time.Duration(timeout),
			Interval:          time.Duration(interval),
			SkipTLSVerify:     tlsVerify,
			FollowRedirects:   followRedirects,
			ForceAttemptHTTP2: forceAttemptHTTP2,
		}
		settings.Headers = headers
		test := httptest.Test{}
		test.Init(&settings, &exportedDataChan, &wg, &logger, debug)
		go test.RunTest()
	}

	// Wait for tests
	wg.Wait()

	// Get results from channel
	var results [][]*httptest.Result
	for i := 0; i < len(urls); i++ {
		incomingData := <-exportedDataChan
		results = append(results, incomingData)
	}

	// EXPORTING TO MODULES

	// define elasticsearch module first
	elasticSearchModule := elasticsearch.Module{}
	elasticSearchModule.Init(&wg, &logger, debug)

	if ipstackConfig.UseIPStack {

		ipstackModule := ipstack.Module{}
		ipstackModule.Init(&wg, &logger, debug)

		logger.Info("Starting IPStack module (ipstack.com)")
		if elConfig.Export {
			mapping := `{
				"properties" : {
				  "ipstack" : {
					"properties": {
					  "LatitudeLongitude" : {
						"type" : "geo_point"
					  }
					}
				}
			  }
			}\n`
			elasticSearchModule.CreateIndex(&elConfig)
			elasticSearchModule.CreateIndexWithMapping(&elConfig, &mapping)
		}
		for i := 0; i < len(urls); i++ {
			if debug {
				logger.Debug(fmt.Sprint("Getting IP information for url ", urls[i]))
			}
			wg.Add(1)
			go ipstackModule.ParseResults(&ipstackConfig, results[i])
		}
		wg.Wait()
		logger.Info("IPStack module completed.")
	}

	// Elasticsearch
	if elConfig.Export || elConfig.ExportToFile {
		logger.Info("Starting ElasticExporter")
		// Start goroutines for each url/endpoint
		for i := 0; i < len(urls); i++ {
			if debug {
				logger.Debug(fmt.Sprintf("Exporting data for url %s", urls[i]))
			}
			wg.Add(1)
			go elasticSearchModule.ExportData(&elConfig, results[i])
		}
		wg.Wait()
		logger.Info("Exporting complete")
	}
}
