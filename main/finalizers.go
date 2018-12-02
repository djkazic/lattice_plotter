package main

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/valyala/bytebufferpool"
	"os"
	"strconv"
)

func writeData(ind int, nonce int, hash string) {
	if db != nil {
		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		key := calcKVPlacement(strNonce, segment)
		err := db.Put(key, []byte(hash), nil)
		if err != nil {
			// Could not update tx
			panic(err)
		}
	}
}

func validateData(ind int, nonce int, hash string) {
	if db != nil {
		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		key := calcKVPlacement(strNonce, segment)
		valBytes, err := db.Get(key, nil)
		if err != nil && err != leveldb.ErrNotFound {
			panic(err)
		}
		strVal := string(valBytes)
		if strVal != hash {
			quitNow.Set()
			fmt.Printf("Error detected on line %d of bucket %s!\n", nonce+1, string(key))
			fmt.Printf("actual: %s\n", strVal)
			fmt.Printf("wanted: %s\n", hash)
			os.Exit(1)
		}
	}
}

func calcKVPlacement(strNonce, segment string) ([]byte) {
	buf := bytebufferpool.Get()
	_, _ = buf.WriteString(segment)
	_, _ = buf.Write(slashBytes)
	_, _ = buf.WriteString(strNonce)
	res := buf.B
	bytebufferpool.Put(buf)

	return res
}
