package ipstack

import (
	"encoding/json"
	"fmt"
	"http-bomber/elasticsearch"
	"http-bomber/httptest"
	"http-bomber/logging"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

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

// Settings ...
type Settings struct {
	APIKey     string
	UseIPStack bool
	Timeout    int
}

// LocationLanguages ...
type LocationLanguages struct {
	Code   string
	Name   string
	Native string
}

// Location ...
type Location struct {
	GeoNameID               int
	Capital                 string
	Languages               []LocationLanguages
	CountryFlag             string
	CountryFlagEmojiUnicode string
	CallingCode             string
	IsEU                    bool
}

// Timezone ...
type Timezone struct {
	ID               string
	CurrentTime      time.Time
	GMTOffset        int
	Code             string
	IsDaylightSaving bool
}

// Currency ...
type Currency struct {
	Code         string
	Name         string
	Plural       string
	Symbol       string
	SymbolNative string
}

// Connection ...
type Connection struct {
	ASN int
	ISP string
}

// Security ...
type Security struct {
	IsProxy     bool
	ProxyType   string
	IsCrawler   bool
	CrawlerName string
	CrawlerType string
	IsTOR       bool
	ThreatLevel string
	ThreadTypes []string
}

// Response ...
type Response struct {
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
	Location          Location
	Timezone          Timezone
	Currency          Currency
	Connection        Connection
	Security          Security
}

// Start ...
func (mod *Module) Start(settings *Settings, results [][]*httptest.Result, elasticModule *elasticsearch.Module, elasticConfig *elasticsearch.Config) {

	if settings.UseIPStack {
		mod.Logger.Info("Starting IPStack module (ipstack.com)")
		if elasticConfig.Export {
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
			elasticModule.CreateIndex(elasticConfig)
			elasticModule.CreateIndexWithMapping(elasticConfig, &mapping)
		}
		for i := 0; i < len(results); i++ {
			if mod.Debug {
				mod.Logger.Debug(fmt.Sprint("Getting IP information for url ", results[i][0].URL))
			}
			mod.WaitGroup.Add(1)
			go mod.ParseResults(settings, results[i])
		}
		mod.WaitGroup.Wait()
		mod.Logger.Info("IPStack module completed.")
	}

}

// ParseResults parses the resultsets from http-bomber
func (mod *Module) ParseResults(settings *Settings, resultSet []*httptest.Result) {
	// map unique IP addresses to IPStack API responses
	m := make(map[string]Response)

	// Check if unique IP addresses are found in results and get IP info for each
	for _, r := range resultSet {
		r.MakeModulesMap()
		if v, found := m[r.DestinationIP]; found {
			r.Modules["ipstack"] = v
		} else {
			response := mod.GetIPInfo(r.DestinationIP, settings)
			m[r.DestinationIP] = response
			r.Modules["ipstack"] = response
		}
	}
	mod.WaitGroup.Done()
}

// GetIPInfo ...
func (mod *Module) GetIPInfo(ipAddr string, settings *Settings) Response {

	response := Response{}
	client := http.Client{
		Timeout: time.Duration(settings.Timeout) * time.Second,
	}
	url := fmt.Sprintf("http://api.ipstack.com/%s?access_key=%s", ipAddr, settings.APIKey)
	resp, err := client.Get(url)
	if err != nil {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprint("Could not get IP information from IPStack ", err))
		}
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if mod.Debug {
			mod.Logger.Debug(fmt.Sprint("Could not get response ", err))
		}
	}
	json.Unmarshal(bodyBytes, &response)
	response.LatitudeLongitude = fmt.Sprintf("%v,%v", response.Latitude, response.Longitude)

	return response
}
