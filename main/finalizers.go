package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func writeData(ind int, nonce int, hashList *[]string) {
	if db != nil {
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
		readSplit[slot] = (*hashList)[ind]
		readStr := strings.Join(readSplit, "\n")
		err := db.WriteString(strKey, readStr)
		if err != nil {
			// Could not update tx
			panic(err)
		}
	}
}

func validateData(ind int, nonce int, hashList *[]string) {
	if db != nil {
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
		hash := (*hashList)[ind]
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

func calcKVPlacement(strNonce, segment string) (string, int) {
	var strBuf strings.Builder

	endInd := len(strNonce) - 1
	if endInd <= 0 {
		endInd = 1
	}
	strBucket := strNonce[:endInd]
	strBuf.WriteString(segment)
	strBuf.Write(slashBytes)
	lastChar := strNonce[endInd:]
	if len(strNonce) <= 1 {
		lastChar = strNonce
		strBuf.Write(zeroStrBytes)
	} else {
		strBuf.WriteString(strBucket)
	}
	slot, err := strconv.Atoi(lastChar)
	if err != nil {
		fmt.Println("Cannot parse slot for db insert")
		panic(err)
	}
	res := strBuf.String()
	strBuf.Reset()

	return res, slot
}
