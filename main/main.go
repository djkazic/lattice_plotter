package main

import (
	"fmt"
	"time"
	"runtime"
	"os"
)

func main() {
	fmt.Println("Starting lattice plotter v0.9.9")

	initMaps()          // Initialize handler map and output map
	warmCache()         // Setup lookup table
	setupGracefulStop() // Start graceful stop goroutine
	parseFlags()        // Parse flags

	fmt.Printf("Address set: %s\n", address)
	baseDir = fmt.Sprintf(baseDir, address)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		os.MkdirAll(baseDir, 0755)
	}
	endNumPlots := 2147483647

	checkBaseDir()      // Check status of base directory
	openDB()			// Badger setup
	getNonceCount()     // Get nonceCount

	// Set max workers for compute
	maxWorkers = runtime.NumCPU() - 1
	if maxWorkers == 0 {
		maxWorkers = 1
	}

	// Start
	start := time.Now()
	if verifyPlots {
		endNumPlots = numExistingPlots
	} else {
		// Prefer the smaller of shortestLen or startPoint
		decideStartPoint()
	}
	// Main loop
	for i := startPoint; i < endNumPlots; i++ {
		if quitNow.IsSet() {
			break
		}
		processPlots(i)
	}
	end := time.Since(start)
	fmt.Printf("Plotter runtime: %s\n", end)
}

func decideStartPoint() {
	if startPoint <= 0 || shortestLen < startPoint {
		fmt.Println("Setting resume point for plot generation")
		startPoint = shortestLen
	}
}
