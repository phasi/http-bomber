package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var elConfig ElasticsearchConfig

type ElasticsearchConfig struct {
	URL            string
	IndexName      string
	Export         bool
	ExportToFile   bool
	ExportFilePath string
}

// bulkExportToElastic is the newer export function
// that can export results in bulk and/or save them to a file
func ElasticExporter(wg *sync.WaitGroup, config *ElasticsearchConfig, resultSet []*Result) {

	var requestData string

	// Make data and write to file
	for _, v := range resultSet {
		data, err := json.Marshal(v)
		if err != nil {
			if Debug {
				DebugLogger.Println("Failed to process JSON")
			}
			return
		}
		requestData += fmt.Sprintf("{ \"index\" : { \"_index\" : \"%s\" } }\n", config.IndexName)
		requestData += string(data)
		requestData += "\n"
	}

	// Create connection pool to add performance
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	// create new http client (TODO: Make timeout configurable)
	client := http.Client{
		Timeout:   2 * time.Second,
		Transport: t,
	}
	// Make a request to elasticsearch
	resp, err := client.Post(fmt.Sprintf("%s/_bulk?pretty", config.URL), "application/x-ndjson", bytes.NewBuffer([]byte(requestData)))
	if err != nil {
		// Log error and append to failed requests, move on to the next row
		if Debug {
			DebugLogger.Println("Failed to send request to Elasticsearch: ", err)
		}
		// Take a little rest since we had an error
		time.Sleep(1 * time.Second)
	}

	// Close response body
	defer resp.Body.Close()
	if strings.HasPrefix(resp.Status, "20") == false {
		if Debug {
			DebugLogger.Printf("Failed to send data to Elasticsearch (Status: %s)", resp.Status)
		}
	}

	if config.ExportToFile {
		randomFileName := fmt.Sprintf("%s-%v", config.ExportFilePath, rand.Int())
		resultFile, err := os.OpenFile(randomFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			InfoLogger.Fatal("Cannot write output to a file.")
		}
		defer resultFile.Close()
		resultFile.WriteString(requestData)
	}

	// GOroutine done
	wg.Done()
}
