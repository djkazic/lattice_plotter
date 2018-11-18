package main

import (
	"github.com/peterbourgon/diskv"
	"strings"
)

func slashTransform(key string) *diskv.PathKey {
	path := strings.Split(key, "/")
	last := len(path) - 1
	return &diskv.PathKey{
		Path:     path[:last],
		FileName: path[last] + ".txt",
	}
}

func inverseSlashTransform(pathKey *diskv.PathKey) string {
	txt := pathKey.FileName[len(pathKey.FileName)-4:]
	if txt != ".txt" {
		panic("Invalid file found in storage folder!")
	}
	return strings.Join(pathKey.Path, "/") + pathKey.FileName[:len(pathKey.FileName)-4]
}

func openDB() {
	db = diskv.New(diskv.Options{
		BasePath:          baseDir,
		AdvancedTransform: slashTransform,
		InverseTransform:  inverseSlashTransform,
		CacheSizeMax:      1024 * 1024,
	})
}
