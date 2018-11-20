package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/magical/argon2"
	"github.com/peterbourgon/diskv"
	"github.com/phf/go-queue/queue"
	"github.com/tevino/abool"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"github.com/mr-tron/base58"
)

type WriteDataParams struct {
	ind   int
	hash  string
	nonce int
}

type ProcessDataParams struct {
	ind   int
	hash  string
	nonce int
}

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
	slashBytes 		 = []byte("/")
)

var (
	verifyPlots = false
	quitNow *abool.AtomicBool
	gracefulStop = make(chan os.Signal)
)

var (
	db       *diskv.Diskv

	hashMut  sync.RWMutex
	hashMap  map[string][][]byte
	indMut   sync.Mutex
	indTable []string

	hashList []string
	prQueue  queue.Queue
	maxWorkers int
)

func initMaps() {
	hashMap = make(map[string][][]byte)
}

func warmCache() {
	// Allocate indTable
	for i := 0; i < 4096; i++ {
		indTable = append(indTable, "")
	}
	// Fill indTable
	for i := 0; i < 4096; i++ {
		strPrefix := fmt.Sprintf("%04d", i)
		indTable[i] = strPrefix
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
		fmt.Println("Data flushed! Cleaning up")
	}()
}

func cachedPrefixLookup(ind int) string {
	var prefix string

	if val := indTable[ind]; val == "" {
		cacheVal := fmt.Sprintf("%04d", ind)
		indMut.Lock()
		indTable[ind] = cacheVal
		indMut.Unlock()
		prefix = cacheVal
	} else {
		prefix = val
	}

	return prefix
}

func calcHash(input []byte) []byte {
	hash, err := argon2.Key(input, input, 1, 2, 1024, 32)
	if err != nil {
		fmt.Printf("Error processing hash for input %x\n", input)
		panic(err)
	}
	return hash
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
		} else {
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
