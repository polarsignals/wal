//go:build wasm

package fs

import "os"

func syncDir(dir string) error {
	// TODO(asubiotto): Issue syncing dirs on wasm.
	return nil
}

func prealloc(_ *os.File, _ int64, _ bool) error {
	// fileutil.Prealloc relies on unix-only functionality which is unavailable
	// on wasm.
	return nil
}
