//go:build wasm

// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0

package metadb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/polarsignals/wal/types"
)

// This file is the BoltMetaDB implementation for the wasm build target. It is
// a mock store not safe to be used in production.
// TODO(asubiotto): For our use-case, BoltDB is overkill. Eventually we should
// just move to something simpler but still correct which can also be used when
// compiled to WASM.

// FileName is the default file name for the bolt db file.
const FileName = "wal-meta.db"

var (
	// ErrUnintialized is returned when any call is made before Load has opened
	// the DB file.
	ErrUnintialized = errors.New("uninitialized")
)

type BoltMetaDB struct {
	dir string
	f   *os.File
}

func (db *BoltMetaDB) ensureOpen(dir string) error {
	if db.dir != "" && db.dir != dir {
		return fmt.Errorf("can't load dir %s, already open in dir %s", dir, db.dir)
	}
	if db.f != nil {
		return nil
	}

	fileName := filepath.Join(dir, FileName)

	open := func() error {
		f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", FileName, err)
		}
		db.dir = dir
		db.f = f
		return nil
	}

	//  1. Check if file exits already. If yes, skip init and just open it.
	//  2. Delete any existing DB file with tmp name
	//  3. Creat a new BoltDB that is empty and has the buckets with a temp name.
	//  4. Once that's committed, rename to final name and Fsync parent dir
	_, err := os.Stat(fileName)
	if err == nil {
		// File exists, just open it
		return open()
	}
	if !errors.Is(err, os.ErrNotExist) {
		// Unknown err just return that
		return fmt.Errorf("failed to stat %s: %w", FileName, err)
	}

	// File doesn't exist, initialize a new DB in a crash-safe way
	if err := safeInitBoltDB(dir); err != nil {
		return fmt.Errorf("failed initializing meta DB: %w", err)
	}

	// All good, now open it!
	return open()
}

func safeInitBoltDB(dir string) error {
	// Delete any old attempts to init that were unsuccessful
	f, err := os.Create(filepath.Join(dir, FileName))
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// And Fsync that parent dir to make sure the new new file with it's new name
	// is persisted!
	// TODO(asubiotto): Directory fsyncs are not supported in WASM.
	/*dirF, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = dirF.Sync()
	closeErr := dirF.Close()
	if err != nil {
		return err
	}
	return closeErr*/
	return nil
}

// Load loads the existing persisted state. If there is no existing state
// implementations are expected to create initialize new storage and return an
// empty state.
func (db *BoltMetaDB) Load(dir string) (types.PersistentState, error) {
	var state types.PersistentState

	if err := db.ensureOpen(dir); err != nil {
		return state, err
	}

	raw, err := io.ReadAll(db.f)
	if err != nil {
		return state, err
	}

	if len(raw) == 0 {
		// Valid state, just an "empty" log.
		return state, nil
	}

	if err := json.Unmarshal(raw, &state); err != nil {
		return state, fmt.Errorf("%w: failed to parse persisted state: %s", types.ErrCorrupt, err)
	}
	return state, nil
}

// CommitState must atomically replace all persisted metadata in the current
// store with the set provided. It must not return until the data is persisted
// durably and in a crash-safe way otherwise the guarantees of the WAL will be
// compromised. The WAL will only ever call this in a single thread at one
// time and it will never be called concurrently with Load however it may be
// called concurrently with Get/SetStable operations.
func (db *BoltMetaDB) CommitState(state types.PersistentState) error {
	if db.f == nil {
		return ErrUnintialized
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to encode persisted state: %w", err)
	}

	// This is not really safe, but good enough for testing.
	if err := db.f.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}

	if _, err := db.f.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	if _, err := db.f.Write(raw); err != nil {
		return fmt.Errorf("failed to write persisted state: %w", err)
	}
	return nil
}

// Close implements io.Closer
func (db *BoltMetaDB) Close() error {
	if db.f == nil {
		return nil
	}
	closeErr := db.f.Close()
	db.f = nil
	return closeErr
}
