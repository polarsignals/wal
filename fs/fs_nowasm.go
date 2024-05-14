//go:build !wasm

package fs

import (
	"os"

	"github.com/coreos/etcd/pkg/fileutil"
)

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func prealloc(f *os.File, sizeInBytes int64, extendFile bool) error {
	return fileutil.Preallocate(f, sizeInBytes, extendFile)
}
