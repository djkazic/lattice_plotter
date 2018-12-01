package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ivpusic/grpool"
	"github.com/orcaman/concurrent-map"
	"github.com/phf/go-queue/queue"
	"runtime"
	"strconv"
	"sync"
	"time"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/sync/semaphore"
	"context"
)

var (
	plotStart time.Time
	plotEnd time.Duration

	cacheMap cmap.ConcurrentMap
	indTable [4096]string
	subPool = grpool.NewPool(runtime.NumCPU() * 8, runtime.NumCPU())
)

func processPlots(nonce int) {
	var (
		commit     bool
		hashList   []string
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
	plotStart = time.Now()
	startingHash := CalcHash(buf.B)
	bytebufferpool.Put(buf)

	// Populate tree from root
	childQueue.Init()
	childQueue.PushBack(startingHash)

	// Iterative approach instead of recursive: we save on memory overhead
	for hashMap.Count() < totalNodes {
		computeNode(&childQueue, &hashMap)
	}

	prQueue.Init()
	serializeHashes(&prQueue, &hashList, startingHash, &hashMap) // Post process hashMap -> hashList
	commit = (nonce + 1) % 10 == 0 && nonce != 0

	if !quitNow.IsSet() {
		if verifyPlots {
			// Validate for hashList
			var validateWg sync.WaitGroup
			validateCtx := context.TODO()
			validateSem := semaphore.NewWeighted(512)
			validateWg.Add(len(hashList))
			for ind, hash := range hashList {
				if err := validateSem.Acquire(validateCtx, 1); err != nil {
					fmt.Println(err)
					break
				}
				go validateData(ind, nonce, hash, &validateWg, validateSem)
			}
			validateWg.Wait()
			plotEnd = time.Since(plotStart)
			fmt.Printf("Nonce %s verified in %s\n", strNonce, plotEnd)
		} else if minePlots {
			// Write for hashList
			var writeWg sync.WaitGroup
			writeCtx := context.TODO()
			writeSem := semaphore.NewWeighted(512)
			writeWg.Add(len(hashList))
			if commit {
				fmt.Println("==============================")
				fmt.Println("Committing nonces to disk")
				fmt.Println("==============================")
				for ind, hash := range hashList {
					if err := writeSem.Acquire(writeCtx, 1); err != nil {
						fmt.Println(err)
						break
					}
					go writeData(ind, nonce, hash, commit, &writeWg, writeSem)
				}
				writeWg.Wait()
				initMaps()
			} else {
				for ind, hash := range hashList {
					writeData(ind, nonce, hash, commit, &writeWg, nil)
				}
			}
			plotEnd = time.Since(plotStart)
			fmt.Printf("Nonce %s timing: %s\n", strNonce, plotEnd)
		}
	}

	// Clear hashList slice
	hashList = nil

	// If plot count exceeds shortestLen, update counter
	if minePlots && nonce > shortestLen && commit {
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

func serializeHashes(prQueue *queue.Queue, hashList *[]string, currHash []byte, hashMap *cmap.ConcurrentMap) {
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
		serializeHashes(prQueue, hashList, head, hashMap)
	}
}
