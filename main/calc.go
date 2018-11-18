package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"
)

func processPlots(nonce int) {
	var computeWg sync.WaitGroup
	var concatBuffer bytes.Buffer

	strNonce := strconv.Itoa(nonce)
	concatBuffer.WriteString(address)
	concatBuffer.WriteString(strNonce)
	startingHash := calcHash(concatBuffer.Bytes())
	concatBuffer.Reset()

	start := time.Now()
	computeNode(startingHash, &computeWg)    // Populate tree from root
	computeWg.Wait()

	serializeHashes(&hashList, startingHash) // Post process hashMap -> hashList
	prQueue.Init()                           // Reset processQueue

	if !quitNow {
		if verifyPlots {
			// Parallel validate for hashList
			for ind, hash := range hashList {
				pdp := &ProcessDataParams{ind, hash, nonce}
				validateData(pdp)
			}
			end := time.Since(start)
			if quitNow { return }
			fmt.Printf("Nonce %s verified in %s\n", strNonce, end)
		} else {
			if quitNow { return }
			// Parallel write for hashList
			for ind, hash := range hashList {
				pdp := &ProcessDataParams{ind, hash, nonce}
				writeData(pdp)
			}
			end := time.Since(start)
			if quitNow { return }
			fmt.Printf("Nonce %s timing: %s\n", strNonce, end)
		}
	}

	// If plot count exceeds shortestLen, update counter
	if !verifyPlots && nonce > shortestLen {
		incrementNonceCt(nonce)
	}

	// Empty hashList slice
	hashList = hashList[:0]
}

func computeNode(root []byte, computeWg *sync.WaitGroup) {
	var hashBuf bytes.Buffer

	// 12 layers under root = 4095 nodes
	hashMut.Lock()
	if len(hashMap) < totalNodes {
		var childrenCalcWg sync.WaitGroup
		var leftHash []byte
		var rightHash []byte
		var rootHash = string(root)

		childrenCalcWg.Add(1)
		calcChildren(root, &hashBuf, &leftHash, &rightHash, &childrenCalcWg)
		childrenCalcWg.Wait()

		// Store children in hashMap
		hashMap[rootHash] = [][]byte{leftHash, rightHash}
		hashMut.Unlock()

		// Spawn goroutines for calculating left and right children
		computeWg.Add(2)
		go runCompute(leftHash, computeWg)
		go runCompute(rightHash, computeWg)
	} else {
		hashMut.Unlock()
	}
}

func calcChildren(rootBytes []byte, hashBuf *bytes.Buffer, leftHash *[]byte, rightHash *[]byte, wg *sync.WaitGroup) {
	// Calculate left node hash
	hashBuf.Write(rootBytes)
	hashBuf.Write(zeroStrBytes)
	*leftHash = calcHash(hashBuf.Bytes())
	hashBuf.Reset()

	// Calculate right node hash
	hashBuf.Write(rootBytes)
	hashBuf.Write(oneStrBytes)
	*rightHash = calcHash(hashBuf.Bytes())
	hashBuf.Reset()
	wg.Done()
}

func runCompute(root []byte, computeWg *sync.WaitGroup) {
	computeNode(root, computeWg)
	computeWg.Done()
}

func serializeHashes(hashList *[]string, currHash []byte) {
	currHashStr := hex.EncodeToString(currHash)
	*hashList = append(*hashList, currHashStr)
	hashMut.Lock()
	if val, ok := hashMap[currHashStr]; ok {
		delete(hashMap, currHashStr)
		hashMut.Unlock()
		val0Str := hex.EncodeToString(val[0])
		val1Str := hex.EncodeToString(val[1])
		prQueue.PushBack(val0Str)
		prQueue.PushBack(val1Str)
	} else {
		hashMut.Unlock()
	}
	for len(*hashList) < totalNodes && prQueue.Len() > 0 {
		head := prQueue.PopFront().([]byte)
		serializeHashes(hashList, head)
	}
}
