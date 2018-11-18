package main

import (
	"flag"
	"fmt"
	"os"
)

func parseFlags() {
	startCmd := flag.NewFlagSet("mine", flag.ExitOnError)
	parsedAddress := startCmd.String("address", "", "Address to generate plots for")
	mineStart := startCmd.Int("start", -1, "Line number in plot to start")
	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
	verifyAddress := verifyCmd.String("address", "", "Address to verify plots with")
	verifyStart := verifyCmd.Int("start", -1, "Line number in plot to start verify")

	argsNeeded := 4
	if len(os.Args) < argsNeeded {
		printUsage()
		return
	}
	verifyPlots = os.Args[1] == "verify"
	if !verifyPlots {
		err := startCmd.Parse(os.Args[2:])
		if err != nil {
			printUsage()
			return
		}
	}
	err := verifyCmd.Parse(os.Args[2:])
	if err != nil {
		printUsage()
		return
	}

	if !verifyPlots {
		if !startCmd.Parsed() || *parsedAddress == "" {
			fmt.Println("Please enter a valid address to mine plots for")
		}
		address = *parsedAddress
		if len(os.Args) > argsNeeded {
			if *mineStart != -1 {
				// parseFlags
				startPoint = *mineStart
				fmt.Printf("Mine start point set to %d\n", startPoint)
			}
		}
	} else {
		if !verifyCmd.Parsed() || *verifyAddress == "" {
			fmt.Println("Please enter a valid address to verify plots for")
		}
		address = *verifyAddress
		// If we're going to verify, we'll also (optionally) need another argument
		if len(os.Args) > argsNeeded {
			if *verifyStart != -1 {
				startPoint = *verifyStart
				// Print out start point + 1 so end-user don't see nonceNum
				fmt.Printf("Verify start point set to nonce %d\n", startPoint)
			}
		}
	}
	if !validateAddress(address) {
		fmt.Println("Please enter a valid address to mine plots for")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  mine   -address <address> [-start <line>]   Start mining")
	fmt.Println("  verify -address <address> [-start <line>]   Verify existing plots")
	os.Exit(0)
}
