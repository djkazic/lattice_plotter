package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting lattice plotter v0.9.7")

	initMaps()          // Initialize handler map and output map
	warmCache()         // Setup lookup table
	setupGracefulStop() // Start graceful stop goroutine
	parseFlags()        // Parse flags

	fmt.Printf("Address set: %s\n", address)
	baseDir = fmt.Sprintf(baseDir, address)
	numPlotsToMake := int64(2147483647)
	endNumPlots := int(numPlotsToMake)

	checkBaseDir()      // Check status of base directory
	openDB()			// Badger setup
	getNonceCount()     // Get nonceCount

	start := time.Now()
	if verifyPlots {
		endNumPlots = numExistingPlots
	} else {
		// Prefer the smaller of shortestLen or startPoint
		decideStartPoint()
	}
	for i := startPoint; i < endNumPlots; i++ {
		if quitNow {
			break
		}
		processPlots(i)
	}
	end := time.Since(start)
	fmt.Printf("%d plots made in %s\n", numPlotsToMake, end)
}

func decideStartPoint() {
	if startPoint <= 0 || shortestLen < startPoint {
		fmt.Println("Setting resume point for plot generation")
		startPoint = shortestLen
	}
}
