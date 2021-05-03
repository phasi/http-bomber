package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"http-bomber/httptest"
	"http-bomber/logging"
)

// Config holds configuration for exporting to elasticsearch
type Config struct {
	URL            string
	IndexName      string
	Export         bool
	ExportToFile   bool
	ExportFilePath string
}

// Module ...
type Module struct {
	WaitGroup *sync.WaitGroup
	Logger    *logging.Logger
	Debug     bool
}

// Init ...
func (mod *Module) Init(wg *sync.WaitGroup, logger *logging.Logger, debug bool) {
	mod.WaitGroup = wg
	mod.Logger = logger
	mod.Debug = debug
}

// Start ...
func (mod *Module) Start(config *Config, results [][]*httptest.Result) {
	if config.Export || config.ExportToFile {
		mod.Logger.Info("Starting Elastic Exporter")
		// Start goroutines for each url/endpoint
		for i := 0; i < len(results); i++ {
			if mod.Debug {
				mod.Logger.Debug(fmt.Sprintf("Exporting data for url %s", results[i][0].URL))
			}
			mod.WaitGroup.Add(1)
			go mod.ExportData(config, results[i])
		}
		mod.WaitGroup.Wait()
		mod.Logger.Info("Exporting complete")
	}
}

// ExportData exports data to either elasticsearch or file or both
func (mod *Module) ExportData(config *Config, resultSet []*httptest.Result) {

	var requestData string

	// Make data and write to file
	for _, v := range resultSet {
		data, err := json.Marshal(v)
		if err != nil {
			if mod.Debug {
				mod.Logger.Debug("Failed to process JSON")
			}
			return
		}

		requestData += fmt.Sprintf("{ \"index\" : { \"_index\" : \"%s\" } }\n", config.IndexName)
		requestData += string(data)
		requestData += "\n"
	}

	if config.Export {
		// Create connection pool to add performance
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.MaxIdleConns = 100
		t.MaxConnsPerHost = 100
		t.MaxIdleConnsPerHost = 100
		// create new http client (TODO: Make timeout configurable)
		client := http.Client{
			Timeout:   5 * time.Second,
			Transport: t,
		}
		// Make a request to elasticsearch
		resp, err := client.Post(fmt.Sprintf("%s/_bulk?pretty", config.URL), "application/x-ndjson", bytes.NewBuffer([]byte(requestData)))
		if err != nil {
			// Log error and append to failed requests, move on to the next row
			if mod.Debug {
				mod.Logger.Debug(fmt.Sprint("Failed to send request to Elasticsearch: ", err))
			}
			// Take a little rest since we had an error
			time.Sleep(1 * time.Second)
		}

		// Close response body
		defer resp.Body.Close()
		if strings.HasPrefix(resp.Status, "20") == false {
			if mod.Debug {
				mod.Logger.Debug(fmt.Sprintf("Failed to send data to Elasticsearch (Status: %s)", resp.Status))
			}
		}
	}

	if config.ExportToFile {
		randomFileName := fmt.Sprintf("%s-%v", config.ExportFilePath, rand.Int())
		resultFile, err := os.OpenFile(randomFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			mod.Logger.Info("Cannot write output to a file.")
		}
		defer resultFile.Close()
		resultFile.WriteString(requestData)
	}

	// GOroutine done
	mod.WaitGroup.Done()
}

// CreateIndex creates an empty index to elasticsearch
func (mod *Module) CreateIndex(config *Config) {
	// create new http client (TODO: Make timeout configurable)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	indexURL := fmt.Sprintf("%s/%s", config.URL, config.IndexName)
	req, err := http.NewRequest("PUT", indexURL, nil)
	if err != nil {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprint("Could form the request:", err))
		}
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprint("Failed to create index:", err))
		}
		return
	}
	// Close response body
	defer resp.Body.Close()
	if strings.HasPrefix(resp.Status, "20") == false {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprintf("Failed to create index (Status: %s)", resp.Status))
			bodyByte, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				mod.Logger.Debug(fmt.Sprint(err))
				return
			}
			mod.Logger.Debug(string(bodyByte))
		}
	}
}

// CreateIndexWithMapping creates an empty index with mapping
func (mod *Module) CreateIndexWithMapping(config *Config, mapping *string) {

	// create new http client (TODO: Make timeout configurable)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	mappingEndpointURL := fmt.Sprintf("%s/%s/_mapping", config.URL, config.IndexName)
	mappingBytes := []byte(*mapping)
	req, err := http.NewRequest("PUT", mappingEndpointURL, bytes.NewBuffer([]byte(mappingBytes)))
	if err != nil {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprint("Could form the request:", err))
		}
		return
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprint("Failed to create mapping:", err))
		}
		return
	}
	// Close response body
	defer resp.Body.Close()
	if strings.HasPrefix(resp.Status, "20") == false {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprintf("Failed to send data to Elasticsearch (Status: %s)", resp.Status))
			bodyByte, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				mod.Logger.Debug(fmt.Sprint(err))
				return
			}
			mod.Logger.Debug(string(bodyByte))
		}
	}
}
