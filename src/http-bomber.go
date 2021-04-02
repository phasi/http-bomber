package main

// Imports
import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// GLOBALS
var InfoLogger *log.Logger
var DebugLogger *log.Logger
var Debug bool
var showVersion bool = false
var url string
var hdrs string
var duration int
var timeout int
var headers http.Header = make(http.Header)
var logFilePath string = "./httptest.log"
var exportedDataChan chan []*Result
var networkStack string = "tcp"

// Wait group for goroutines
var wg sync.WaitGroup

// Settings holds information on one HTTP test
type Settings struct {
	URL      string
	Headers  http.Header
	Duration time.Duration
	Timeout  time.Duration
}

// Result holds information on one request
type Result struct {
	Timestamp       time.Time       `json:"@timestamp"`
	URL             string          `json:"url"`
	ReqHeaders      http.Header     `json:"req_headers"`
	RespHeaders     http.Header     `json:"resp_headers"`
	DestinationIP   string          `json:"destination_ip"`
	DestinationPort int             `json:"destination_port"`
	RespStatusCode  int             `json:"resp_status_code"`
	ReqStartTime    time.Time       `json:"req_start_time"`
	ReqEndTime      time.Time       `json:"req_end_time"`
	ReqRoundTrip    time.Duration   `json:"req_round_trip"`
	IPStack         IPStackResponse `json:"ipstack"`
}

// Configure application logging
func configLogging() {

	var mw io.Writer
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("Could not open logfile")
		mw = io.Writer(os.Stdout)
	} else {
		mw = io.MultiWriter(os.Stdout, logFile)
	}
	InfoLogger = log.New(mw, "", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(mw, "DEBUG ", log.Ldate|log.Ltime|log.Lshortfile)
}

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

	// FLAGS

	// Version info
	flag.BoolVar(&showVersion, "version", false, "Show version info")

	// Common flags
	flag.BoolVar(&Debug, "debug", false, "This flag turns debugging on.")
	flag.StringVar(&networkStack, "n", "tcp4", "Network stack")
	flag.StringVar(&url, "url", "http://localhost", "URL to test. Add multiple URLs separated by a comma (no whitespaces in between)")
	flag.StringVar(&hdrs, "headers", fmt.Sprintf("X-Tested-With:http-bomber/%s", AppVersion), "Additional headers example-> Host:localhost,X-Custom-Header:helloworld")
	flag.IntVar(&duration, "duration", 10, "Test duration in seconds")
	flag.IntVar(&timeout, "timeout", 5, "Connection timeout in seconds")

	// MODULE FLAGS

	// Elasticsearch
	flag.StringVar(&elConfig.URL, "elastic-url", "http://localhost:9200", "Elastic search URL")
	flag.StringVar(&elConfig.IndexName, "elastic-index", "testdata", "Elasticsearch index name")
	flag.BoolVar(&elConfig.Export, "elastic-export", false, "Export data to elasticsearch")
	flag.BoolVar(&elConfig.ExportToFile, "elastic-export-to-file", false, "Export data to file in elasticsearch format")
	flag.StringVar(&elConfig.ExportFilePath, "elastic-export-filepath", "/tmp/http-bomber-results.json", "Specify filepath for Elasticsearch export")

	// IPStack
	flag.BoolVar(&IPStackConfig.UseIPStack, "ipstack", false, "Use IPStack for example for getting geolocation details")
	flag.StringVar(&IPStackConfig.APIKey, "ipstack-apikey", "1234", "Your personal IPStack API key")
	flag.IntVar(&IPStackConfig.Timeout, "ipstack-timeout", 3, "IPStack connect timeout")
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
	InfoLogger.Println("Starting HTTP Bomber", AppVersion)

	// Get URLs
	urls := strings.Split(url, ",")
	// Make channel for results
	exportedDataChan = make(chan []*Result, len(urls))
	// Set the number of wait groups based on the quantity of URLs provided by the user
	wg.Add(len(urls))

	// Goroutines for each url provided
	for i := 0; i < len(urls); i++ {
		InfoLogger.Printf("Starting test %v (URL: %s)", i+1, urls[i])
		settings := Settings{URL: urls[i], Duration: time.Duration(duration), Timeout: time.Duration(timeout)}
		settings.Headers = headers
		go RunTest(&settings, &wg)
	}

	// Wait for tests
	wg.Wait()

	// Get results from channel
	var results [][]*Result
	for i := 0; i < len(urls); i++ {
		incomingData := <-exportedDataChan
		results = append(results, incomingData)
	}

	// EXPORTING TO MODULES

	if IPStackConfig.UseIPStack {
		InfoLogger.Println("Starting IPStack module (ipstack.com)")
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
			ElasticCreateIndex(&elConfig)
			ElasticCreateIndexWithMapping(&elConfig, &mapping)
		}
		for i := 0; i < len(urls); i++ {
			if Debug {
				DebugLogger.Println("Getting IP information for url", urls[i])
			}
			wg.Add(1)
			go IPStackParseResults(&wg, results[i])
		}
		wg.Wait()
		InfoLogger.Println("IPStack module completed.")
	}

	// Elasticsearch
	if elConfig.Export || elConfig.ExportToFile {
		InfoLogger.Println("Starting ElasticExporter")
		// Start goroutines for each url/endpoint
		for i := 0; i < len(urls); i++ {
			if Debug {
				DebugLogger.Println("Exporting data for url", urls[i])
			}
			wg.Add(1)
			go ElasticExporter(&wg, &elConfig, results[i])
		}
		wg.Wait()
		InfoLogger.Println("Exporting complete")
	}

}
