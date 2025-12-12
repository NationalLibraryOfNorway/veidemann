// Copyright © 2019 National Library of Norway
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package importutil

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/ristretto/v2/z"
	"github.com/rs/zerolog/log"
)

// badgerLogger is a log adapter that implements badger.Logger
type badgerLogger struct {
	prefix string
}

func (l badgerLogger) Errorf(fmt string, args ...interface{}) {
	log.Error().Msgf(l.prefix+fmt, args...)
}

func (l badgerLogger) Warningf(fmt string, args ...interface{}) {
	log.Warn().Msgf(l.prefix+fmt, args...)
}

func (l badgerLogger) Infof(fmt string, args ...interface{}) {
	log.Debug().Msgf(l.prefix+fmt, args...)
}

func (l badgerLogger) Debugf(fmt string, args ...interface{}) {
	log.Trace().Msgf(l.prefix+fmt, args...)
}

type ExistsCode int

const (
	Undefined ExistsCode = iota
	NewKey
	NewId
	Exists
)

func (e ExistsCode) ExistsInVeidemann() bool {
	return e > NewKey
}

func (e ExistsCode) String() string {
	if e < Undefined || e > Exists {
		return "UNKNOWN"
	}
	names := []string{
		"UNDEFINED",
		"NEW_KEY",
		"NEW_ID",
		"EXISTS"}

	return names[e]
}

type KeyNormalizer interface {
	Normalize(key string) (string, error)
}

type ImportDb struct {
	db *badger.DB
	gc *time.Ticker
}

func NewImportDb(dbDir string, truncate bool) (*ImportDb, error) {
	if err := os.MkdirAll(dbDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to create db dir %s: %w", dbDir, err)
	}
	opts := badger.DefaultOptions(dbDir)
	opts.Logger = badgerLogger{prefix: "Badger: "}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open db %s: %w", dbDir, err)
	}

	if truncate {
		err = db.DropAll()
		if err != nil {
			return nil, fmt.Errorf("failed to reset db %s: %w", dbDir, err)
		}
	}

	d := &ImportDb{
		db: db,
		gc: time.NewTicker(5 * time.Minute),
	}

	// Run GC in background
	go func() {
		for range d.gc.C {
			d.RunValueLogGC(0.7)
		}
	}()

	return d, nil
}

func (d *ImportDb) RunValueLogGC(discardRatio float64) {
	var err error
	for err == nil {
		err = d.db.RunValueLogGC(discardRatio)
	}
}

// Close closes the database, stops the GC ticker and waits for
func (d *ImportDb) Close() {
	_ = d.db.RunValueLogGC(0.7)
	d.gc.Stop()
	_ = d.db.Close()
}

func ImportExisting(db *ImportDb, client configV1.ConfigClient, kind configV1.Kind, keyNormalizer KeyNormalizer) error {
	req := &configV1.ListRequest{
		Kind: kind,
	}
	r, err := client.ListConfigObjects(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to list %s from Veidemann: %w", kind.String(), err)
	}

	var count, imported, failed int

	start := time.Now()
	for {
		msg, err := r.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading %s from Veidemann: %w", kind.String(), err)
		}

		id := msg.GetId()
		key := msg.GetMeta().GetName()

		if keyNormalizer != nil {
			key, err = keyNormalizer.Normalize(key)
			if err != nil {
				failed++
				log.Error().Err(err).Str("key", msg.GetMeta().GetName()).Str("id", id).Msg("Normalization failed")
				continue
			}
		}

		code, _, err := db.Set(key, id)
		if err != nil {
			return fmt.Errorf("error writing to db: %w", err)
		}

		l := log.With().Str("key", key).Str("id", id).Str("kind", kind.String()).Logger()

		count++
		switch code {
		case Undefined:
			failed++
		case NewKey:
			imported++
			l.Info().Msg("New key imported from Veidemann")
		case NewId:
			l.Info().Msg("New id imported from Veidemann")
		case Exists:
			l.Info().Msg("Already imported from Veidemann")
		}
	}
	elapsed := time.Since(start)
	log.Info().Str("kind", kind.String()).Int("total", count).Int("imported", imported).Int("errors", failed).Str("elapsed", elapsed.String()).Msg("Import from Veidemann complete")

	return nil
}

// Iterate iterates over all keys in the db and calls the function with the key and value.
// The function is not called in parallel.
func (d *ImportDb) Iterate(fn func([]byte, []byte)) error {
	stream := d.db.NewStream()

	// -- Optional settings
	stream.NumGo = 16                     // Set number of goroutines to use for iteration.
	stream.LogPrefix = "Badger.Streaming" // For identifying stream logs. Outputs to Logger.
	// -- End of optional settings.

	// Send is called serially, while Stream.Orchestrate is running.
	stream.Send = func(buf *z.Buffer) error {
		list, err := badger.BufferToKVList(buf)
		if err != nil {
			return err
		}
		for _, kv := range list.GetKv() {
			fn(kv.Key, kv.Value)
		}
		return nil
	}

	// Run the stream
	return stream.Orchestrate(context.Background())
}

// Get returns the ids for the key
func (d *ImportDb) Get(key string) (ids []string, err error) {
	for {
		err = d.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(key))
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			if err != nil {
				return err
			}

			return item.Value(func(v []byte) error {
				ids = d.bytesToStringArray(v)
				return nil
			})
		})
		if !errors.Is(err, badger.ErrConflict) {
			break
		}
	}
	return ids, err
}

// Set sets the id as a value for the key.
func (d *ImportDb) Set(key string, id string) (code ExistsCode, ids []string, err error) {
	for {
		err = d.db.Update(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(key))
			if errors.Is(err, badger.ErrKeyNotFound) {
				code = NewKey
				ids = append(ids, id)
				v := d.stringArrayToBytes(ids)
				return txn.Set([]byte(key), v)
			}
			if err != nil {
				return err
			}

			err = item.Value(func(v []byte) error {
				ids = d.bytesToStringArray(v)
				return nil
			})
			if err != nil {
				return err
			}
			if !stringArrayContains(ids, id) {
				code = NewId
				ids = append(ids, id)
				v := d.stringArrayToBytes(ids)
				return txn.Set([]byte(key), v)
			} else {
				code = Exists
			}
			return nil
		})
		if !errors.Is(err, badger.ErrConflict) {
			break
		}
	}
	return
}

// stringArrayToBytes returns a byte array from a string array
func (d *ImportDb) stringArrayToBytes(v []string) []byte {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(v); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// bytesToStringArray returns a string array from a byte array
func (d *ImportDb) bytesToStringArray(v []byte) []string {
	buf := bytes.NewBuffer(v)
	strs := []string{}
	if err := gob.NewDecoder(buf).Decode(&strs); err != nil && !errors.Is(err, io.EOF) {
		panic(err)
	}
	return strs
}

// stringArrayContains returns true if the string array contains the string
func stringArrayContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
