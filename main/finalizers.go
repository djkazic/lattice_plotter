package main

import (
	"errors"
	"fmt"
	"github.com/valyala/bytebufferpool"
	"os"
	"strconv"
	"strings"
)

func writeData(payload interface{}) interface{} {
	if db != nil {
		pdp := payload.(*WriteDataParams)
		ind := pdp.ind
		hash := *pdp.hash
		nonce := pdp.nonce

		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		strKey, slot := calcKVPlacement(strNonce, segment)
		readGroup := db.ReadString(strKey)
		readSplit := strings.Split(readGroup, "\n")
		if len(readSplit) < 10 {
			neededSlots := 10 - len(readSplit)
			for i := 0; i < neededSlots; i++ {
				readSplit = append(readSplit, "")
			}
		}
		readSplit[slot] = hash
		readStr := strings.Join(readSplit, "\n")
		err := db.WriteString(strKey, readStr)
		if err != nil {
			// Could not update tx
			panic(err)
		}
	}
	return nil
}

func validateData(payload interface{}) interface{} {
	if db != nil {
		pdp := payload.(*ReadDataParams)
		ind := pdp.ind
		hash := *pdp.hash
		nonce := pdp.nonce

		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		strKey, slot := calcKVPlacement(strNonce, segment)
		readGroup := db.ReadString(strKey)
		readSplit := strings.Split(readGroup, "\n")
		if len(readSplit) < slot {
			err := errors.New("could not locate slot in readSplit of db")
			panic(err)
		}
		strVal := readSplit[slot]
		if strVal != hash {
			quitNow.Set()
			fmt.Printf("Error detected on line %d of bucket %s!\n", nonce+1, strKey)
			fmt.Printf("actual: %s\n", strVal)
			fmt.Printf("wanted: %s\n", hash)
			fmt.Printf("index: %d\n", ind)
			os.Exit(1)
		}
	}
	return nil
}

func calcKVPlacement(strNonce, segment string) (string, int) {
	var res string
	buffer := bytebufferpool.Get()

	endInd := len(strNonce) - 1
	if endInd <= 0 {
		endInd = 1
	}
	strBucket := strNonce[:endInd]

	_, err := buffer.WriteString(segment)
	if err != nil {
		fmt.Printf("error: calcKV first buffer write failed! %v\n", err)
	}
	_, err = buffer.Write(slashBytes)
	if err != nil {
		fmt.Printf("error: calcKV second buffer write failed! %v\n", err)
	}
	lastChar := strNonce[endInd:]
	if len(strNonce) <= 1 {
		lastChar = strNonce
		_, err = buffer.Write(zeroStrBytes)
		if err != nil {
			fmt.Printf("error: calcKV final buffer write failed! %v\n", err)
		}
	} else {
		_, err = buffer.WriteString(strBucket)
		if err != nil {
			fmt.Printf("error: calcKV final buffer write failed! %v\n", err)
		}
	}
	slot, err := strconv.Atoi(lastChar)
	if err != nil {
		fmt.Println("Cannot parse slot for db insert")
		panic(err)
	}
	res = buffer.String()
	buffer.Reset()
	bytebufferpool.Put(buffer)

	return res, slot
}
