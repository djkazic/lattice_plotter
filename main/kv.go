package main

import (
	"github.com/syndtr/goleveldb/leveldb"
)

func openDB() {
	var err error
	db, err = leveldb.OpenFile(baseDir, nil)
	if err != nil {
		panic(err)
	}
}
