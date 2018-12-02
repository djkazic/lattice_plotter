package main

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"time"
)

func openDB() {
	var err error
	db, err = leveldb.OpenFile(baseDir, nil)
	if err != nil {
		if errors.IsCorrupted(err) {
			db, err = leveldb.RecoverFile(baseDir, nil)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
	// In case compaction is triggered
	time.Sleep(time.Second)
}
