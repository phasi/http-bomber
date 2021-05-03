package httptest

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync"
	"time"

	"http-bomber/logging"
)

// Settings holds information on one HTTP test
type Settings struct {
	URL               string
	Headers           http.Header
	Duration          time.Duration
	Timeout           time.Duration
	Interval          time.Duration
	NetworkStack      string
	SkipTLSVerify     bool
	FollowRedirects   bool
	ForceAttemptHTTP2 bool
}

// Result holds information on one request
type Result struct {
	Timestamp       time.Time              `json:"@timestamp"`
	URL             string                 `json:"url"`
	ReqHeaders      http.Header            `json:"req_headers"`
	RespHeaders     http.Header            `json:"resp_headers"`
	DestinationIP   string                 `json:"destination_ip"`
	DestinationPort int                    `json:"destination_port"`
	RespStatusCode  int                    `json:"resp_status_code"`
	ReqStartTime    time.Time              `json:"req_start_time"`
	ReqEndTime      time.Time              `json:"req_end_time"`
	ReqRoundTrip    time.Duration          `json:"req_round_trip"`
	Modules         map[string]interface{} `json:"modules"`
}

// MakeModulesMap initializes the Modules map. Should be executed by custom modules before trying to add data
func (r *Result) MakeModulesMap() {
	if r.Modules == nil {
		r.Modules = make(map[string]interface{})
	}
}

// Test ...
type Test struct {
	Settings         Settings
	ExportedDataChan *chan []*Result
	WaitGroup        *sync.WaitGroup
	Debug            bool
	Logger           *logging.Logger
}

// Init ...
func (test *Test) Init(settings *Settings, exportedDataChan *chan []*Result, wg *sync.WaitGroup, logger *logging.Logger, debug bool) {
	test.Settings = *settings
	test.ExportedDataChan = exportedDataChan
	test.WaitGroup = wg
	test.Debug = debug
	test.Logger = logger
}

// Start runs the test
// results are appended in a resultset ([]Result) which is then passed on to the channel
func (test *Test) Start() {
	var resultSet []*Result

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Dial = (func(network, addr string) (net.Conn, error) {
		return (&net.Dialer{
			Timeout:   3 * time.Second,
			LocalAddr: nil,
			DualStack: false,
		}).Dial(test.Settings.NetworkStack, addr)
	})
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: test.Settings.SkipTLSVerify}
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.ForceAttemptHTTP2 = test.Settings.ForceAttemptHTTP2 // make optional later

	client := http.Client{
		Timeout:   test.Settings.Timeout * time.Second,
		Transport: t,
	}

	// check if client should follow redirects
	if !test.Settings.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	startTime := time.Now()
	for {
		if time.Since(startTime) >= test.Settings.Duration*time.Second {
			break
		}
		result := test.makeRequest(&client)
		if result != nil {
			resultSet = append(resultSet, result)
		}
		time.Sleep(test.Settings.Interval * time.Millisecond)
	}
	// Pass resultset to channel
	*test.ExportedDataChan <- resultSet
	// Let our program know that this goroutine is done :-)
	test.WaitGroup.Done()
}

// Make a single HTTP request as per settings object
func (test *Test) makeRequest(client *http.Client) *Result {

	req, err := http.NewRequest("GET", test.Settings.URL, nil)
	if err != nil {
		if test.Debug {
			test.Logger.Debug(fmt.Sprint("Failed to form request: ", err))
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
	req.Header = test.Settings.Headers
	r := &Result{URL: test.Settings.URL, ReqStartTime: time.Now()}
	resp, err := client.Do(req)
	if err != nil {
		if test.Debug {
			test.Logger.Debug(fmt.Sprint("Failed request: ", err))
		}
		return nil
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		if test.Debug {
			test.Logger.Debug(fmt.Sprint(err))
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
	if test.Debug {
		test.Logger.Debug(fmt.Sprint(r.URL, " ", r.RespStatusCode, r.ReqRoundTrip))
	}
	r.Timestamp = time.Now()
	return r
}
