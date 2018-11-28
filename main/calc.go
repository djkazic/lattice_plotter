package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ivpusic/grpool"
	"github.com/orcaman/concurrent-map"
	"github.com/phf/go-queue/queue"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	plotStart time.Time
	plotEnd time.Duration

	hashMap  cmap.ConcurrentMap
	indTable [4096]string
	subPool = grpool.NewPool(runtime.NumCPU() * 8, runtime.NumCPU())
)

func processPlots(nonce int) {
	var (
		strBuf strings.Builder
		hashList   []string
		childQueue queue.Queue
		prQueue    queue.Queue
	)

	// Calculate this nonce's starting hash
	strNonce := strconv.Itoa(nonce)
	strBuf.WriteString(address)
	strBuf.WriteString(strNonce)
	plotStart = time.Now()
	calcBytes := []byte(strBuf.String())
	startingHash := calcHash(calcBytes)
	strBuf.Reset()

	// Populate tree from root
	childQueue.Init()
	childQueue.PushBack(startingHash)

	// Iterative approach instead of recursive: we save on memory overhead
	for hashMap.Count() < totalNodes {
		computeNode(&childQueue)
	}

	prQueue.Init()
	serializeHashes(&prQueue, &hashList, startingHash) // Post process hashMap -> hashList

	if !quitNow.IsSet() {
		if verifyPlots {
			// Validate for hashList
			for ind := range hashList {
				validateData(ind, nonce, &hashList)
			}
			plotEnd = time.Since(plotStart)
			fmt.Printf("Nonce %s verified in %s\n", strNonce, plotEnd)
		} else {
			// Write for hashList
			for ind := range hashList {
				writeData(ind, nonce, &hashList)
			}
			plotEnd = time.Since(plotStart)
			fmt.Printf("Nonce %s timing: %s\n", strNonce, plotEnd)
		}
	}

	// Clear hashList slice
	hashList = nil

	// If plot count exceeds shortestLen, update counter
	if !verifyPlots && nonce > shortestLen {
		incrementNonceCt(nonce)
	}
}

func computeNode(chpQueue *queue.Queue) {
	// 12 layers under root = 4095 nodes
	var leftHash, rightHash []byte

	if hashMap.Count() < totalNodes {
		root := chpQueue.PopFront().([]byte)

		// Block until children node hashes calculated
		calcChildren(root, &leftHash, &rightHash)

		// Store children hashes in hashMap
		childHashes := [][]byte{leftHash, rightHash}
		rootStr := hex.EncodeToString(root)
		hashMap.Set(rootStr, childHashes)

		// Return sub-nodes for processing
		chpQueue.PushBack(leftHash)
		chpQueue.PushBack(rightHash)
	}
}

func calcChildren(root []byte, leftHash *[]byte, rightHash *[]byte) {
	subPool.WaitCount(2)
	subPool.JobQueue <- func() {
		calcSubNode(root, zeroStrBytes, leftHash)
		subPool.JobDone()
	}
	subPool.JobQueue <- func() {
		calcSubNode(root, oneStrBytes, rightHash)
		subPool.JobDone()
	}
	subPool.WaitAll()
}

func calcSubNode(root []byte, instruct []byte, target *[]byte) {
	var inputBytes [33]byte

	copy(inputBytes[:32], root)
	copy(inputBytes[32:], instruct)
	*target = calcHash(inputBytes[:])
}

func serializeHashes(prQueue *queue.Queue, hashList *[]string, currHash []byte) {
	currHashStr := hex.EncodeToString(currHash)
	*hashList = append(*hashList, currHashStr)
	tmp, ok := hashMap.Get(currHashStr)
	if ok && tmp != nil {
		val := tmp.([][]byte)
		prQueue.PushBack(val[0])
		prQueue.PushBack(val[1])
		hashMap.Remove(currHashStr)
	}
	for prQueue.Len() > 0 && len(*hashList) < totalNodes {
		head := prQueue.PopFront().([]byte)
		serializeHashes(prQueue, hashList, head)
	}
}
