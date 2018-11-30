package main

import (
	"github.com/orcaman/concurrent-map"
	"github.com/phf/go-queue/queue"
	"testing"
)

func BenchmarkComputeNode(b *testing.B) {
	hashMap := cmap.New()
	startHashStr := "Yellow Submarine"
	startHash := CalcHash([]byte(startHashStr))
	testQueue := queue.New()
	testQueue.PushBack(startHash)

	b.ReportAllocs()
	b.ResetTimer()
	for hashMap.Count() < b.N {
		computeNode(testQueue, &hashMap)
	}
}
