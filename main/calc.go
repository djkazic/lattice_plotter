package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

func processPlots(nonce int) {
	var strNonce string

	var concatBuffer bytes.Buffer

	var start time.Time
	var end time.Duration

	var pdp *ProcessDataParams
	var wdp *WriteDataParams

	strNonce = strconv.Itoa(nonce)
	concatBuffer.WriteString(address)
	concatBuffer.WriteString(strNonce)
	startingHash := calcHash(concatBuffer.Bytes())
	concatBuffer.Reset()

	start = time.Now()
	computeNode(startingHash)                // Populate tree from root

	prQueue.Init()                           // Reset processQueue
	serializeHashes(&hashList, startingHash) // Post process hashMap -> hashList

	if !quitNow.IsSet() {
		if verifyPlots {
			// Validate for hashList
			for ind := range hashList {
				pdp = &ProcessDataParams{ind, hashList[ind], nonce}
				validateData(pdp)
			}
			end = time.Since(start)
			fmt.Printf("Nonce %s verified in %s\n", strNonce, end)
		} else {
			// Write for hashList
			for ind := range hashList {
				wdp = &WriteDataParams{ind, hashList[ind], nonce}
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

func computeNode(root []byte) {
	// 12 layers under root = 4095 nodes
	var hmLen int
	var leftHash, rightHash []byte
	var rootHash = hex.EncodeToString(root)

	hmLen = hashMap.Len()

	if hmLen < totalNodes {
		// Block until children node hashes calculated
		childHashes := calcChildren(root, &leftHash, &rightHash)

		// Store children hashes in hashMap
		hashMap.Set(rootHash[:16], childHashes)

		// Compute sub-nodes for left and right children
		computeNode(leftHash)
		computeNode(rightHash)
	}
}

func calcChildren(rootBytes []byte, leftHash *[]byte, rightHash *[]byte) [][]byte {
	var hashBuf bytes.Buffer

	hashBuf.Write(rootBytes)
	hashBuf.Write(zeroStrBytes)
	*leftHash = calcHash(hashBuf.Bytes())
	hashBuf.Reset()
	hashBuf.Write(rootBytes)
	hashBuf.Write(oneStrBytes)
	*rightHash = calcHash(hashBuf.Bytes())
	hashBuf.Reset()

	return [][]byte{*leftHash, *rightHash}
}

func serializeHashes(hashList *[]string, currHash []byte) {
	currHashStr := hex.EncodeToString(currHash)
	*hashList = append(*hashList, currHashStr)
	tmp, ok := hashMap.Get(currHashStr[:16])

	if ok && tmp != nil {
		val := tmp.([][]byte)
		prQueue.PushBack(val[0])
		prQueue.PushBack(val[1])
		hashMap.Del(currHashStr[:16])
	}
	for len(*hashList) < totalNodes && prQueue.Len() > 0 {
		head := prQueue.PopFront().([]byte)
		serializeHashes(hashList, head)
	}
}
