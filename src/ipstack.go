package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// IPStackConfig ...
var IPStackConfig IPStackUserSettings

// IPStackUserSettings ...
type IPStackUserSettings struct {
	APIKey     string
	UseIPStack bool
	Timeout    int
}

// IPStackLocationLanguages ...
type IPStackLocationLanguages struct {
	Code   string
	Name   string
	Native string
}

// IPStackLocation ...
type IPStackLocation struct {
	GeoNameID               int
	Capital                 string
	Languages               []IPStackLocationLanguages
	CountryFlag             string
	CountryFlagEmojiUnicode string
	CallingCode             string
	IsEU                    bool
}

// IPStackTimezone ...
type IPStackTimezone struct {
	ID               string
	CurrentTime      time.Time
	GMTOffset        int
	Code             string
	IsDaylightSaving bool
}

// IPStackCurrency ...
type IPStackCurrency struct {
	Code         string
	Name         string
	Plural       string
	Symbol       string
	SymbolNative string
}

// IPStackConnection ...
type IPStackConnection struct {
	ASN int
	ISP string
}

// IPStackSecurity ...
type IPStackSecurity struct {
	IsProxy     bool
	ProxyType   string
	IsCrawler   bool
	CrawlerName string
	CrawlerType string
	IsTOR       bool
	ThreatLevel string
	ThreadTypes []string
}

// IPStackResponse ...
type IPStackResponse struct {
	IP                string
	Hostname          string
	Type              string
	ContinentCode     string
	ContinentName     string
	CountryCode       string
	CountryName       string
	RegionCode        string
	RegionName        string
	City              string
	Zip               string
	Latitude          float64
	Longitude         float64
	LatitudeLongitude string
	Location          IPStackLocation
	Timezone          IPStackTimezone
	Currency          IPStackCurrency
	Connection        IPStackConnection
	Security          IPStackSecurity
}

// GetIPInfo ...
func (response *IPStackResponse) GetIPInfo(ipAddr string) IPStackResponse {

	client := http.Client{
		Timeout: time.Duration(IPStackConfig.Timeout) * time.Second,
	}
	url := fmt.Sprintf("http://api.ipstack.com/%s?access_key=%s", ipAddr, IPStackConfig.APIKey)
	resp, err := client.Get(url)
	if err != nil {
		if Debug {
			DebugLogger.Println("Could not get IP information from IPStack", err)
		}
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if Debug {
			DebugLogger.Println("Could not get response", err)
		}
	}
	json.Unmarshal(bodyBytes, &response)
	response.LatitudeLongitude = fmt.Sprintf("%v,%v", response.Latitude, response.Longitude)

	return *response
}

// IPStackParseResults parses the resultsets from http-bomber
func IPStackParseResults(wg *sync.WaitGroup, resultSet []*Result) {
	// map unique IP addresses to IPStack API responses
	m := make(map[string]IPStackResponse)

	// Check if unique IP addresses are found in results and get IP info for each
	for _, r := range resultSet {
		if v, found := m[r.DestinationIP]; found {
			r.IPStack = v
		} else {
			stack := IPStackResponse{}
			stack.GetIPInfo(r.DestinationIP)
			m[r.DestinationIP] = stack
			r.IPStack = stack
		}
	}
	wg.Done()
}
