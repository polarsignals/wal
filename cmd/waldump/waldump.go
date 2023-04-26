// Copyright (c) HashiCorp, Inc.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/polarsignals/wal/fs"
	"github.com/polarsignals/wal/segment"
	"github.com/polarsignals/wal/types"
)

type opts struct {
	Dir    string
	After  uint64
	Before uint64
}

func main() {
	var o opts
	flag.Uint64Var(&o.After, "after", 0, "specified an index to use as an exclusive lower bound when dumping log entries.")
	flag.Uint64Var(&o.Before, "before", 0, "specified an index to use as an exclusive upper bound when dumping log entries.")

	flag.Parse()

	// Accept dir as positional arg
	o.Dir = flag.Arg(0)
	if o.Dir == "" {
		fmt.Println("Usage: waldump [-after INDEX] [-before INDEX] <path to WAL dir>")
		os.Exit(1)
	}

	vfs := fs.New()
	f := segment.NewFiler(o.Dir, vfs)

	err := f.DumpLogs(o.After, o.Before, func(info types.SegmentInfo, e types.LogEntry) (bool, error) {
		fmt.Println("TODO: decode bytes:", e.Data)
		return true, nil
	})
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}
}
