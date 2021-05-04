module http-bomber

require "http-bomber/httptest" v0.0.0
require "http-bomber/logging" v0.0.0
require "http-bomber/elasticsearch" v0.0.0
require "http-bomber/ipstack" v0.0.0


replace http-bomber/httptest => ./httptest
replace http-bomber/logging => ./logging
replace http-bomber/elasticsearch => ./elasticsearch
replace http-bomber/ipstack => ./ipstack
go 1.16
