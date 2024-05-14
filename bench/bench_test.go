// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/polarsignals/wal"
	"github.com/polarsignals/wal/types"
)

func BenchmarkAppend(b *testing.B) {
	sizes := []int{
		10,
		1024,
		100 * 1024,
		1024 * 1024,
	}
	sizeNames := []string{
		"10",
		"1k",
		"100k",
		"1m",
	}
	batchSizes := []int{1, 10}

	for i, s := range sizes {
		for _, bSize := range batchSizes {
			b.Run(fmt.Sprintf("entrySize=%s/batchSize=%d/v=WAL", sizeNames[i], bSize), func(b *testing.B) {
				ls, done := openWAL(b)
				defer done()
				// close _first_ (defers run in reverse order) before done() which will
				// delete since rotate could still be happening
				defer ls.Close()
				runAppendBench(b, ls, s, bSize)
			})
		}
	}
}

func openWAL(b *testing.B) (*wal.WAL, func()) {
	tmpDir, err := os.MkdirTemp("", "raft-wal-bench-*")
	require.NoError(b, err)

	// Force every 1k append to create a new segment to profile segment rotation.
	ls, err := wal.Open(tmpDir, wal.WithSegmentSize(512))
	require.NoError(b, err)

	return ls, func() { os.RemoveAll(tmpDir) }
}

func runAppendBench(b *testing.B, ls wal.LogStore, s, n int) {
	// Pre-create batch, we'll just adjust the indexes in the loop
	batch := make([]types.LogEntry, n)
	for i := range batch {
		batch[i] = types.LogEntry{
			Data: randomData[:s],
		}
	}

	b.ResetTimer()
	idx := uint64(1)
	for i := 0; i < b.N; i++ {
		for j := range batch {
			batch[j].Index = idx
			idx++
		}
		b.StartTimer()
		err := ls.StoreLogs(batch)
		b.StopTimer()
		if err != nil {
			b.Fatalf("error appending: %s", err)
		}
	}
}

func BenchmarkGetLogs(b *testing.B) {
	sizes := []int{
		1000,
		1_000_000,
	}
	sizeNames := []string{
		"1k",
		"1m",
	}
	for i, s := range sizes {
		func() {
			wLs, done := openWAL(b)
			defer done()
			// close _first_ (defers run in reverse order) before done() which will
			// delete since rotate could still be happening
			defer wLs.Close()
			populateLogs(b, wLs, s, 128) // fixed 128 byte logs

			b.Run(fmt.Sprintf("numLogs=%s/v=WAL", sizeNames[i]), func(b *testing.B) {
				runGetLogBench(b, wLs, s)
			})
		}()
	}
}

func populateLogs(b *testing.B, ls wal.LogStore, n, size int) {
	batchSize := 1000
	batch := make([]types.LogEntry, 0, batchSize)
	start := time.Now()
	for i := 0; i < n; i++ {
		l := types.LogEntry{Index: uint64(i + 1), Data: randomData[:2]}
		batch = append(batch, l)
		if len(batch) == batchSize {
			err := ls.StoreLogs(batch)
			require.NoError(b, err)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		err := ls.StoreLogs(batch)
		require.NoError(b, err)
	}
	b.Logf("populateTime=%s", time.Since(start))
}

func runGetLogBench(b *testing.B, ls wal.LogStore, n int) {
	b.ResetTimer()
	var log types.LogEntry
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		err := ls.GetLog(uint64((i+1)%n), &log)
		b.StopTimer()
		require.NoError(b, err)
	}
}

func BenchmarkOSRename(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "raft-wal-bench-*")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpName := filepath.Join(tmpDir, fmt.Sprintf("%d.tmp", i%2))
		// Create the tmp file outside timer loop to simulate it happening in the
		// background
		f, err := os.OpenFile(tmpName, os.O_CREATE|os.O_EXCL|os.O_RDWR, os.FileMode(0644))
		require.NoError(b, err)
		f.Close()

		fname := filepath.Join(tmpDir, fmt.Sprintf("test-%d.txt", i))
		b.StartTimer()
		// Note we are not fsyncing parent dir in either case
		err = os.Rename(tmpName, fname)
		if err != nil {
			panic(err)
		}
		b.StopTimer()
	}
}
