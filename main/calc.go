package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ivpusic/grpool"
	"github.com/orcaman/concurrent-map"
	"github.com/phf/go-queue/queue"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/valyala/bytebufferpool"
	"strconv"
	"time"
)

var (
	plotStart time.Time
	plotEnd time.Duration

	indTable [4096]string
	subPool  *grpool.Pool
)

func processPlots(nonce int) {
	var (
		hashList   [][]byte
		childQueue queue.Queue
		prQueue    queue.Queue
		hashMap    cmap.ConcurrentMap
	)
	// Init hashMap
	hashMap = cmap.New()

	// Calculate this nonce's starting hash
	strNonce := strconv.Itoa(nonce)
	buf := bytebufferpool.Get()
	_, _ = buf.WriteString(address)
	_, _ = buf.WriteString(strNonce)
	startingHash := CalcHash(buf.B)
	bytebufferpool.Put(buf)

	// Populate tree from root
	childQueue.Init()
	childQueue.PushBack(startingHash)
	plotStart = time.Now()

	// Iterative approach instead of recursive: we save on memory overhead
	for hashMap.Count() < totalNodes {
		computeNode(&childQueue, &hashMap)
	}

	prQueue.Init()
	serializeHashes(&prQueue, &hashList, startingHash, &hashMap) // Post process hashMap -> hashList

	if !quitNow.IsSet() {
		if verifyPlots {
			// Validate for hashList
			for ind, hash := range hashList {
				validateData(ind, nonce, hash)
			}
			plotEnd = time.Since(plotStart)
			fmt.Printf("Nonce %s validated: %s\n", strNonce, plotEnd)
		} else if minePlots {
			// Write for hashList
			batch := new(leveldb.Batch)
			for ind, hash := range hashList {
				writeData(ind, nonce, hash, batch)
			}
			err := db.Write(batch, nil)
			if err != nil {
				// Could not update tx
				panic(err)
			}
			plotEnd = time.Since(plotStart)
			fmt.Printf("Nonce %s timing: %s\n", strNonce, plotEnd)
		}
	}

	// Clear hashList slice
	hashList = nil

	// If plot count exceeds shortestLen, update counter
	if minePlots && nonce > shortestLen {
		incrementNonceCt(nonce)
	}
}

func computeNode(chpQueue *queue.Queue, hashMap *cmap.ConcurrentMap) {
	// 12 layers under root = 4095 nodes
	if hashMap.Count() < totalNodes {
		var leftHash, rightHash []byte

		// Block until children node hashes calculated
		root := chpQueue.PopFront().([]byte)
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
	buf := bytebufferpool.Get()
	_, _ = buf.Write(root)
	_, _ = buf.Write(instruct)
	inputBytes := buf.B
	*target = CalcHash(inputBytes)
	bytebufferpool.Put(buf)
}

func serializeHashes(prQueue *queue.Queue, hashList *[][]byte, currHash []byte, hashMap *cmap.ConcurrentMap) {
	currHashStr := hex.EncodeToString(currHash)
	*hashList = append(*hashList, currHash)
	tmp, ok := hashMap.Get(currHashStr)
	if ok && tmp != nil {
		val := tmp.([][]byte)
		prQueue.PushBack(val[0])
		prQueue.PushBack(val[1])
		hashMap.Remove(currHashStr)
	}
	for prQueue.Len() > 0 && len(*hashList) < totalNodes {
		head := prQueue.PopFront().([]byte)
		serializeHashes(prQueue, hashList, head, hashMap)
	}
}
