package main

import (
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkCalcKVPlacement(b *testing.B) {
	var strNonceList []string
	var segmentList []string

	for i := 0; i < b.N; i++ {
		strNonce := strconv.Itoa(i)
		segment := fmt.Sprintf("%04d", i)
		strNonceList = append(strNonceList, strNonce)
		segmentList = append(segmentList, segment)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calcKVPlacement(strNonceList[i], segmentList[i])
	}
}
