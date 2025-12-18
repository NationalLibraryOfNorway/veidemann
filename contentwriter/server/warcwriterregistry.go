/*
 * Copyright 2021 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package server

import (
	"sync"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/database"
	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/writer"
	"github.com/nlnwa/gowarc"
	"github.com/rs/zerolog/log"
)

type WarcWriter interface {
	Write(meta *contentwriterV1.WriteRequestMeta, record ...gowarc.WarcRecord) (*contentwriterV1.WriteReply, error)
	Close() error
}

type warcWriterRegistry struct {
	writerOpts     writer.Options
	dbAdapter      database.ConfigAdapter
	contentAdapter database.ContentAdapter
	warcWriters    map[string]WarcWriter
	lock           sync.Mutex
}

func newWarcWriterRegistry(writerOpts writer.Options, db database.ConfigAdapter, content database.ContentAdapter) *warcWriterRegistry {
	return &warcWriterRegistry{
		writerOpts:     writerOpts,
		warcWriters:    make(map[string]WarcWriter),
		dbAdapter:      db,
		contentAdapter: content,
	}
}

func (w *warcWriterRegistry) GetWarcWriter(collection *configV1.ConfigObject, recordMeta *contentwriterV1.WriteRequestMeta_RecordMeta) WarcWriter {
	w.lock.Lock()
	defer w.lock.Unlock()

	key := collection.GetMeta().GetName() + "#" + recordMeta.GetSubCollection().String()
	if ww, ok := w.warcWriters[key]; ok {
		return ww
	}
	ww := writer.New(w.writerOpts, w.dbAdapter, w.contentAdapter, collection, recordMeta)
	w.warcWriters[key] = ww
	return ww
}

func (w *warcWriterRegistry) Close() {
	w.lock.Lock()
	defer w.lock.Unlock()

	for _, ww := range w.warcWriters {
		err := ww.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close WarcWriter")
		}
	}
}
