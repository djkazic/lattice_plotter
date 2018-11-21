package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

func writeData(payload interface{}) interface{} {
	var buffer bytes.Buffer

	if db != nil {
		pdp := payload.(*WriteDataParams)
		ind := pdp.ind
		hash := *pdp.hash
		nonce := pdp.nonce

		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		endInd := len(strNonce) - 1
		if endInd <= 0 {
			endInd = 1
		}
		strBucket := strNonce[:endInd]
		buffer.WriteString(segment)
		buffer.Write(slashBytes)
		buffer.WriteString(strBucket)
		buffer.Write(slashBytes)
		buffer.WriteString(strNonce)
		strKey := buffer.String()
		buffer.Reset()
		err := db.WriteString(strKey, hash)
		if err != nil {
			// Could not update tx
			panic(err)
		}
	}
	return nil
}

func validateData(payload interface{}) interface{} {
	var buffer bytes.Buffer

	if db != nil {
		pdp := payload.(*ReadDataParams)
		ind := pdp.ind
		hash := *pdp.hash
		nonce := pdp.nonce

		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		endInd := len(strNonce) - 1
		if endInd <= 0 {
			endInd = 1
		}
		strBucket := strNonce[:endInd]
		buffer.WriteString(segment)
		buffer.Write(slashBytes)
		buffer.WriteString(strBucket)
		buffer.Write(slashBytes)
		buffer.WriteString(strNonce)
		strKey := buffer.String()
		buffer.Reset()

		val, err := db.Read(strKey)
		if err != nil {
			fmt.Printf("Error getting key %s\n", strKey)
			panic(err)
		} else {
			strVal := string(val)
			if strVal != hash {
				quitNow.Set()
				fmt.Printf("Error detected on line %d of bucket %s!\n", nonce+1, strKey)
				fmt.Printf("actual: %s\n", strVal)
				fmt.Printf("wanted: %s\n", hash)
				fmt.Printf("index: %d\n", ind)
				os.Exit(1)
			}
		}
	}
	return nil
}
