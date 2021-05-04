package examplemodule

import (
	"fmt"
	"sync"
)

// Config holds configuration for exporting to elasticsearch
type Config struct {
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

	mod.Logger.Info("Starting Example Module")
	// Do something for each resultset
	for i := 0; i < len(results); i++ {
		if mod.Debug {
			mod.Logger.Debug("I am a debug message")
		}
		// Add to wait group
		mod.WaitGroup.Add(1)
		// For performance execute each resultset in its own goroutine
		// REPLACE below line with your custom function
		go fmt.Println(results[i])
	}
	// IMPORTANT: Wait for each goroutine
	mod.WaitGroup.Wait()
	mod.Logger.Info("Example module completed")
}
