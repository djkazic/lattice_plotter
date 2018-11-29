package main

import (
	"errors"
	"fmt"
	"github.com/valyala/bytebufferpool"
	"os"
	"strconv"
	"strings"
)

func writeData(ind int, nonce int, hashList *[]string, checkPointing bool) {
	if db != nil {
		var readGroup string

		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		strKey, slot := calcKVPlacement(strNonce, segment)
		if tmp, ok := cacheMap.Get(strKey); ok {
			readGroup = tmp.(string)
		} else {
			readGroup = db.ReadString(strKey)
		}

		readSplit := strings.Split(readGroup, "\n")
		if len(readSplit) < 10 {
			neededSlots := 10 - len(readSplit)
			for i := 0; i < neededSlots; i++ {
				readSplit = append(readSplit, "")
			}
		}
		readSplit[slot] = (*hashList)[ind]
		readStr := strings.Join(readSplit, "\n")

		if checkPointing {
			err := db.WriteString(strKey, readStr)
			if err != nil {
				// Could not update tx
				panic(err)
			}
			cacheMap.Remove(strKey)
		} else {
			cacheMap.Set(strKey, readStr)
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
			os.Exit(1)
		}
	}
}

func calcKVPlacement(strNonce, segment string) (string, int) {
	buf := bytebufferpool.Get()
	endInd := len(strNonce) - 1
	if endInd <= 0 {
		endInd = 1
	}
	strBucket := strNonce[:endInd]
	buf.WriteString(segment)
	buf.Write(slashBytes)
	lastChar := strNonce[endInd:]
	if len(strNonce) <= 1 {
		lastChar = strNonce
		buf.Write(zeroStrBytes)
	} else {
		buf.WriteString(strBucket)
	}
	slot, err := strconv.Atoi(lastChar)
	if err != nil {
		fmt.Println("Cannot parse slot for db insert")
		panic(err)
	}
	res := buf.String()
	bytebufferpool.Put(buf)

	return res, slot
}
