//go:build windows
// +build windows

// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0

package fs

import (
	"os"

	"github.com/polarsignals/wal/types"
)

var _ types.WritableFile = &File{}

// File wraps an os.File and implements types.WritableFile. It ensures that the
// first time Sync is called on the file, that the parent directory is also
// Fsynced to ensure a crash won't cause the FS to forget the file is there.
//
// Postponing this allows us to ensure that we do the minimum necessary fsyncs
// but still ensure all required fsyncs are done by the time we acknowledge
// committed data in the new file.
type File struct {
	new uint32 // atomically accessed, keep it aligned!
	dir string
	os.File
}

// Sync calls fsync on the underlying file. If this is the first call to Sync
// since creation it also fsyncs the parent dir.
func (f *File) Sync() error {
	// Sync the underlying file
	fileInfo, err := f.File.Stat()
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return nil
	}
	if err := f.File.Sync(); err != nil {
		return err
	}
	return nil
}
