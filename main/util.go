package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/orcaman/concurrent-map"
	"github.com/peterbourgon/diskv"
	"github.com/tevino/abool"
	"golang.org/x/crypto/argon2"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"syscall"
)

const (
	totalNodes = 4095
	addressChecksumLen = 4
)

var (
	address          string
	baseDir = userHomeDir() + "/lattice/plotter/%s/"
	shortestLen      int
	numExistingPlots int
	startPoint       int
	zeroStrBytes     = []byte("0")
	oneStrBytes      = []byte("1")
	slashBytes       = []byte("/")
	dmpMutex         sync.Mutex
	profiling = false
)

var (
	minePlots   = false
	verifyPlots = false
	quitNow *abool.AtomicBool
	gracefulStop = make(chan os.Signal)
)

var (
	db       *diskv.Diskv
	maxWorkers int
)

func initMaps() {
	hashMap = cmap.New()
	cacheMap = cmap.New()
}

func warmCache() {
	// Allocate + fill indTable
	for i := 0; i < 4096; i++ {
		indTable[i] = fmt.Sprintf("%04d", i)
	}
}

func setupGracefulStop() {
	quitNow = abool.New()
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		<-gracefulStop
		fmt.Println("\nWaiting for writeData to finish...")
		quitNow.Set()
		if profiling {
			pprof.StopCPUProfile()
		}
		fmt.Println("Data flushed! Cleaning up")
	}()
}

func cachedPrefixLookup(ind int) string {
	var prefix string

	if val := indTable[ind]; val == "" {
		cacheVal := fmt.Sprintf("%04d", ind)
		indTable[ind] = cacheVal
		prefix = cacheVal
	} else {
		prefix = val
	}

	return prefix
}

func calcHash(input []byte) []byte {
	return argon2.Key(input, input, 1, 1024, 2, 32)
}

func checkBaseDir() {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		fmt.Println("Base directory not found. Creating...")
		os.MkdirAll(baseDir, os.ModePerm)
	}
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

func getNonceCount() {
	firstStart := false
	strKey := "nonceCount"

	val := db.ReadString(strKey)
	if val == "" {
		fmt.Println("Could not read nonceCount")
		firstStart = true
		startPoint = 0
	} else {
		intNonceCount, err := strconv.Atoi(val)
		if err != nil {
			panic(err)
		}
		if verifyPlots {
			numExistingPlots = intNonceCount
			fmt.Printf("endNumPlots = %d\n", numExistingPlots)
		} else if minePlots {
			// If the nonceCount is smaller than the explicitly set startPoint
			shortestLen = intNonceCount
			fmt.Printf("shortestLen = %d\n", shortestLen)
		}
	}
	if !firstStart && val == "" {
		fmt.Println("Error accessing nonceCount value")
	}
}

func incrementNonceCt(nonceCount int) {
	if db != nil && !quitNow.IsSet() {
		strKey := "nonceCount"
		strVal := strconv.Itoa(nonceCount)
		err := db.WriteString(strKey, strVal)
		if err != nil {
			fmt.Println("Cannot increment nonce, terminating")
			panic(err)
		}
	}
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func validateAddress(address string) bool {
	if len(address) < 34 {
		return false
	}
	pubKeyHash, err := base58.Decode(address)
	if err != nil {
		fmt.Println("Could not decode address provided")
		return false
	}
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Equal(actualChecksum, targetChecksum)
}
