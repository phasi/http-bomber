package main

// Imports
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Globals
var InfoLogger *log.Logger
var DebugLogger *log.Logger
var Debug bool
var logFilePath string = "./httptest.log"
var exportedDataChan chan []*Result
var useExporter bool = false
var networkStack string = "tcp"

// Settings holds information on one HTTP test
type Settings struct {
	URL      string
	Headers  http.Header
	Duration time.Duration
	Timeout  time.Duration
}

// Result holds information on one request
type Result struct {
	Timestamp       time.Time     `json:"@timestamp"`
	URL             string        `json:"url"`
	ReqHeaders      http.Header   `json:"req_headers"`
	RespHeaders     http.Header   `json:"resp_headers"`
	DestinationIP   string        `json:"destination_ip"`
	DestinationPort int           `json:"destination_port"`
	RespStatusCode  int           `json:"resp_status_code"`
	ReqStartTime    time.Time     `json:"req_start_time"`
	ReqEndTime      time.Time     `json:"req_end_time"`
	ReqRoundTrip    time.Duration `json:"req_round_trip"`
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

// Make a single HTTP request as per settings object
func makeRequest(client *http.Client, settings *Settings) *Result {

	req, err := http.NewRequest("GET", settings.URL, nil)
	if err != nil {
		if Debug {
			DebugLogger.Println("Failed to form request: ", err)
		}
		return nil
	}

	var rmtaddr string

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			rmtaddr = connInfo.Conn.RemoteAddr().String()
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	req.Header = settings.Headers
	r := &Result{URL: settings.URL, ReqStartTime: time.Now()}
	resp, err := client.Do(req)
	if err != nil {
		if Debug {
			DebugLogger.Println("Failed request: ", err)
		}
		return nil
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		if Debug {
			DebugLogger.Println(err)
		}
		return nil
	}
	r.ReqEndTime = time.Now()
	r.ReqRoundTrip = r.ReqEndTime.Sub(r.ReqStartTime)
	r.RespStatusCode = resp.StatusCode
	r.ReqHeaders = req.Header
	r.RespHeaders = resp.Header
	// separate IP and port
	dst := strings.Split(rmtaddr, ":")
	dstPort, _ := strconv.Atoi(dst[len(dst)-1])
	dstIP := strings.Join(dst[:len(dst)-1], ":")

	r.DestinationIP = dstIP
	r.DestinationPort = dstPort
	if Debug {
		DebugLogger.Println(r.URL, r.RespStatusCode, r.ReqRoundTrip)
	}
	return r
}

// RunTest goroutine executes the test as per settings object
// results are appended in a resultset ([]Result) which is then passed on to the channel
func RunTest(settings *Settings, wg *sync.WaitGroup) {
	var resultSet []*Result

	t := &http.Transport{
		Dial: (func(network, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   3 * time.Second,
				LocalAddr: nil,
				DualStack: false,
			}).Dial(networkStack, addr)
		}),
	}
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.ForceAttemptHTTP2 = false // make optional later

	client := http.Client{
		Timeout:   settings.Timeout * time.Second,
		Transport: t,
	}

	start_time := time.Now()
	for {
		result := makeRequest(&client, settings)
		if result != nil {
			resultSet = append(resultSet, result)
		}
		if time.Since(start_time) >= settings.Duration*time.Second {
			break
		}
		time.Sleep(100)
	}
	// Pass resultset to channel if exporter is used
	if useExporter {
		exportedDataChan <- resultSet
	}
	// Let our program know that this goroutine is done :-)
	wg.Done()
}

// exporter exports test results to elasticsearch
func exporter(wg *sync.WaitGroup, elasticURL *string) {

	// Read one object from channel
	rs := <-exportedDataChan

	// Create connection pool to add performance
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	var failedReqs []*Result
	// iterate over resultset
	for _, el := range rs {
		// set the "processing" timestamp
		el.Timestamp = time.Now()
		// convert data to JSON
		data, err := json.Marshal(el)
		if err != nil {
			if Debug {
				DebugLogger.Println("Failed to process row: ", el)
			}
		}
		// create new http client (TODO: Make timeout configurable)
		client := http.Client{
			Timeout:   2 * time.Second,
			Transport: t,
		}
		// Make a request to elasticsearch
		resp, err := client.Post(*elasticURL, "application/json", bytes.NewBuffer(data))
		if err != nil {
			// Log error and append to failed requests, move on to the next row
			if Debug {
				DebugLogger.Println("Failed to make HTTP request: ", el, err)
			}
			// Take a rest since we had an error
			time.Sleep(1 * time.Second)
			// append the failed request
			failedReqs = append(failedReqs, el)
			// Stop program execution at X failed requests
			if len(failedReqs) > 10 {
				InfoLogger.Fatal("Too many failed requests. Check your network settings and Elasticsearch URL. (for example http://localhost:9200/indexname/_doc)")
			}
			continue
		}
		if strings.HasPrefix(resp.Status, "20") == false {
			// append to failed requests if response status is wrong
			failedReqs = append(failedReqs, el)
			if Debug {
				DebugLogger.Println("Failed to make HTTP request: ", el, "Status: ", resp.Status)
			}
			// Stop program execution at X failed requests
			if len(failedReqs) > 10 {
				InfoLogger.Fatal("Too many failed requests. Check your network settings and Elasticsearch URL. (for example http://localhost:9200/indexname/_doc)")
			}
		}
		// Close response body
		defer resp.Body.Close()
		// Sleep just in case to avoid problems
		time.Sleep(1)
	}
	// Let user know the amount of failed requests per URL
	if Debug {
		DebugLogger.Printf("There were %v failed elasticsearch requests for url %s", len(failedReqs), rs[0].URL)
	}
	// Let our program know that this goroutine is done :-)
	wg.Done()

}

// Initial operations
func init() {
	// configure logging
	configLogging()
}

func parseHeadersFlag(headers *string, parsedHeaders *http.Header) {
	hdrsSlice := strings.Split(*headers, ",")
	for _, v := range hdrsSlice {
		hdr := strings.Split(v, ":")
		parsedHeaders.Add(hdr[0], hdr[1])
	}
}

func main() {

	// Flags
	flag.BoolVar(&Debug, "debug", false, "This flag turns debugging on.")
	flag.StringVar(&networkStack, "n", "tcp4", "Network stack")
	url := flag.String("url", "http://localhost", "URL to test. Add multiple URLs separated by a comma (no whitespaces in between)")
	hdrs := flag.String("headers", fmt.Sprintf("X-Tested-With:http-bomber/%s", AppVersion), "Additional headers example-> Host:localhost,X-Custom-Header:helloworld")
	duration := flag.Int("duration", 10, "Test duration in seconds")
	timeout := flag.Int("timeout", 5, "Connection timeout in seconds")
	var elURL string
	flag.StringVar(&elURL, "el-url", "http://localhost:9200/testdata/_doc", "Elastic search URL")
	flag.BoolVar(&useExporter, "export", false, "Export data to elasticsearch")
	var showVersion bool = false
	flag.BoolVar(&showVersion, "version", false, "Show version info")
	flag.Parse()

	if showVersion {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	var headers http.Header = make(http.Header)
	headers.Add("User-Agent", fmt.Sprintf("http-bomber/%s", AppVersion))
	parseHeadersFlag(hdrs, &headers)

	InfoLogger.Println("Starting HTTP Bomber", AppVersion)

	// Wait group for goroutines
	var wg sync.WaitGroup

	// Get URLs
	urls := strings.Split(*url, ",")
	// Make channel if exporter is used
	if useExporter {
		exportedDataChan = make(chan []*Result, len(urls))
	}
	// Set the number of wait groups
	wg.Add(len(urls))
	// Goroutines for each url provided
	for i := 0; i < len(urls); i++ {
		InfoLogger.Printf("Starting test %v (URL: %s)", i+1, urls[i])
		settings := Settings{URL: urls[i], Duration: time.Duration(*duration), Timeout: time.Duration(*timeout)}
		settings.Headers = headers
		go RunTest(&settings, &wg)
	}

	// Wait for tests
	wg.Wait()

	// Export test results to elasticsearch
	if useExporter {
		InfoLogger.Println("Starting Exporter")
		InfoLogger.Println("Exporting resultsets to elasticsearch")
		// Start goroutines for each url/endpoint
		for i := 0; i < len(urls); i++ {
			wg.Add(1)
			go exporter(&wg, &elURL)
		}
		// Wait for export goroutines
		wg.Wait()
		InfoLogger.Println("Exporting complete")
	}

}
