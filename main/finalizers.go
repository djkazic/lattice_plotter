package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/valyala/bytebufferpool"
	"os"
	"strconv"
)

func writeData(ind int, nonce int, hash []byte, batch *leveldb.Batch) {
	if db != nil {
		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		key := calcKVPlacement(strNonce, segment)
		batch.Put(key, hash)
	}
}

func validateData(ind int, nonce int, hash []byte) {
	if db != nil {
		segment := cachedPrefixLookup(ind)
		strNonce := strconv.Itoa(nonce)
		key := calcKVPlacement(strNonce, segment)
		valBytes, err := db.Get(key, nil)
		if err != nil && err != leveldb.ErrNotFound {
			panic(err)
		}
		if !bytes.Equal(valBytes, hash) {
			quitNow.Set()
			fmt.Printf("Error detected on line %d of bucket %s!\n", nonce+1, string(key))
			fmt.Printf("actual: %s\n", hex.EncodeToString(valBytes))
			fmt.Printf("wanted: %s\n", hex.EncodeToString(hash))
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
