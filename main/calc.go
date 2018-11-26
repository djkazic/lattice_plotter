package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/valyala/bytebufferpool"
	"strconv"
	"sync"
	"time"
)

func processPlots(nonce int) {
	var (
		strNonce string
		concatBuffer bytes.Buffer
		start time.Time
		end time.Duration
		pdp *ReadDataParams
		wdp *WriteDataParams
		leftChild, rightChild []byte
	)

	// Calculate this nonce's starting hash
	strNonce = strconv.Itoa(nonce)
	concatBuffer.WriteString(address)
	concatBuffer.WriteString(strNonce)
	startingHash := calcHash(concatBuffer.Bytes())
	concatBuffer.Reset()

	start = time.Now()

	// Populate tree from root
	childQueue.Init()
	childQueue.PushBack(startingHash)
	quitComputeLoop := false

	// Iterative approach instead of recursive: we save on memory overhead
	for !quitComputeLoop {
		computeInput := childQueue.PopFront().([]byte)
		leftChild, rightChild, quitComputeLoop = computeNode(computeInput)
		if quitComputeLoop {
			break
		}
		childQueue.PushBack(leftChild)
		childQueue.PushBack(rightChild)
	}

	prQueue.Init()                           // Reset processQueue
	serializeHashes(&hashList, startingHash) // Post process hashMap -> hashList

	if !quitNow.IsSet() {
		if verifyPlots {
			// Validate for hashList
			for ind := range hashList {
				pdp = &ReadDataParams{ind, &hashList[ind], nonce}
				validateData(pdp)
			}
			end = time.Since(start)
			fmt.Printf("Nonce %s verified in %s\n", strNonce, end)
		} else {
			// Write for hashList
			for ind := range hashList {
				wdp = &WriteDataParams{ind, &hashList[ind], nonce}
				writeData(wdp)
			}
			end = time.Since(start)
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

func computeNode(root []byte) ([]byte, []byte, bool) {
	// 12 layers under root = 4095 nodes
	var rootHash = hex.EncodeToString(root)
	var leftHash, rightHash []byte
	var childWg sync.WaitGroup

	if hashMap.Len() < totalNodes {
		// Block until children node hashes calculated
		ccp := &CalcChildParams{&root, &leftHash, &rightHash, &childWg}
		childPool.Process(ccp)

		// Store children hashes in hashMap
		childHashes := [][]byte{leftHash, rightHash}
		hashMap.Set(rootHash, childHashes)

		// Return sub-nodes for processing
		return leftHash, rightHash, false
	}
	// Return quit flag as true
	return nil, nil, true
}

func calcChildren(payload interface{}) interface{} {
	var rootPtr *[]byte
	var leftHashPtr, rightHashPtr *[]byte
	var childWg sync.WaitGroup

	ccp := payload.(*CalcChildParams)
	rootPtr = ccp.rootPtr
	leftHashPtr = ccp.leftHash
	rightHashPtr = ccp.rightHash

	childWg.Add(2)
	go calcSubNode(rootPtr, &zeroStrBytes, leftHashPtr, &childWg)
	go calcSubNode(rootPtr, &oneStrBytes, rightHashPtr, &childWg)
	childWg.Wait()

	return nil
}

func calcSubNode(rootPtr *[]byte, instruction *[]byte, target *[]byte, wg *sync.WaitGroup) {
	inputBuf := bytebufferpool.Get()
	_, err := inputBuf.Write(*rootPtr)
	if err != nil {
		fmt.Printf("error: calcSubNode first buffer write failed! %v\n", err)
	}
	_, err = inputBuf.Write(*instruction)
	if err != nil {
		fmt.Printf("error: calcSubNode second buffer write failed! %v\n", err)
	}
	*target = calcHash(inputBuf.B)
	// inputBuf.Reset()
	bytebufferpool.Put(inputBuf)
	wg.Done()
}

func serializeHashes(hashList *[]string, currHash []byte) {
	currHashStr := hex.EncodeToString(currHash)
	*hashList = append(*hashList, currHashStr)
	tmp, ok := hashMap.Get(currHashStr)

	if ok && tmp != nil {
		val := tmp.([][]byte)
		prQueue.PushBack(val[0])
		prQueue.PushBack(val[1])
		hashMap.Del(currHashStr)
	}
	for len(*hashList) < totalNodes && prQueue.Len() > 0 {
		head := prQueue.PopFront().([]byte)
		serializeHashes(hashList, head)
	}
}
