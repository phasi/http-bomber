package main

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Make a single HTTP request as per settings object
func makeRequest(client *http.Client, settings *Settings) *Result {

	req, err := http.NewRequest("GET", settings.URL, nil)
	if err != nil {
		if debug {
			debugLogger.Println("Failed to form request: ", err)
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
		if debug {
			debugLogger.Println("Failed request: ", err)
		}
		return nil
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		if debug {
			debugLogger.Println(err)
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
	if debug {
		debugLogger.Println(r.URL, r.RespStatusCode, r.ReqRoundTrip)
	}
	r.Timestamp = time.Now()
	return r
}

// RunTest goroutine executes the test as per settings object
// results are appended in a resultset ([]Result) which is then passed on to the channel
func RunTest(settings *Settings, wg *sync.WaitGroup) {
	var resultSet []*Result

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Dial = (func(network, addr string) (net.Conn, error) {
		return (&net.Dialer{
			Timeout:   3 * time.Second,
			LocalAddr: nil,
			DualStack: false,
		}).Dial(networkStack, addr)
	})
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: settings.SkipTLSVerify}
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.ForceAttemptHTTP2 = settings.ForceAttemptHTTP2 // make optional later

	client := http.Client{
		Timeout:   settings.Timeout * time.Second,
		Transport: t,
	}

	// check if client should follow redirects
	if !settings.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	startTime := time.Now()
	for {
		if time.Since(startTime) >= settings.Duration*time.Second {
			break
		}
		result := makeRequest(&client, settings)
		if result != nil {
			resultSet = append(resultSet, result)
		}
		time.Sleep(settings.Interval * time.Millisecond)
	}
	// Pass resultset to channel
	exportedDataChan <- resultSet
	// Let our program know that this goroutine is done :-)
	wg.Done()
}
