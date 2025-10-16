// Copyright © 2023 National Library of Norway
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
	"context"
	"encoding/json"
	"io"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config"
	"github.com/rs/zerolog/log"
)

type DuplicateReportRecord struct {
	Name    string
	Records []Record
}

type Record struct {
	Id string
}

type DuplicateKindReporter struct {
	*ImportDb
}

func (d DuplicateKindReporter) Report(w io.Writer) error {
	var nrOfDuplicateKeys int32
	var nrOfDuplicateValues int32
	var rec DuplicateReportRecord

	checkDuplicate := func(k []byte, v []byte) {
		ids := d.bytesToStringArray(v)
		// If there is only one id, it is not a duplicate
		if len(ids) < 2 {
			return
		}
		nrOfDuplicateKeys++
		nrOfDuplicateValues += int32(len(ids))

		rec.Name = string(k)
		rec.Records = rec.Records[:0]
		for _, id := range ids {
			rec.Records = append(rec.Records, Record{Id: id})
		}

		l := log.With().Str("key", rec.Name).Logger()

		b, err := json.Marshal(&rec)
		if err != nil {
			l.Error().Err(err).Msg("failed to marshal record to json")
			return
		}
		if _, err := w.Write(b); err != nil {
			l.Error().Err(err).Msg("failed to write record")
			return
		}
		if _, err := w.Write([]byte{'\n'}); err != nil {
			log.Error().Err(err).Msg("failed to write record")
			return
		}
	}

	err := d.Iterate(checkDuplicate)
	if err != nil {
		return err
	}

	log.Info().Int32("duplicate keys", nrOfDuplicateKeys).Int32("duplicate values", nrOfDuplicateValues).Msg("Duplicate report completed")
	return nil
}

type SeedDuplicateReportRecord struct {
	Host  string
	Seeds []SeedRecord
}

type SeedRecord struct {
	SeedId            string
	Uri               string
	SeedDescription   string
	EntityId          string
	EntityName        string
	EntityDescription string
}

type SeedReporter struct {
	*ImportDb
	Client configV1.ConfigClient
}

func (d SeedReporter) Report(w io.Writer) error {
	var nrOfDuplicateKeys int32
	var nrOfDuplicateValues int32
	rec := &SeedDuplicateReportRecord{}

	checkDuplicate := func(k []byte, v []byte) {
		ids := d.bytesToStringArray(v)
		// If there is only one id, it is not a duplicate
		if len(ids) < 2 {
			return
		}
		nrOfDuplicateKeys++
		nrOfDuplicateValues += int32(len(ids))

		rec.Host = string(k)
		// Avoid memory allocation by reusing the same slice
		rec.Seeds = rec.Seeds[:0]

		l := log.With().Str("key", rec.Host).Logger()

		for _, id := range ids {
			l = l.With().Str("id", id).Logger()
			ref := &configV1.ConfigRef{Id: id, Kind: configV1.Kind_seed}
			seed, err := d.Client.GetConfigObject(context.Background(), ref)
			if err != nil {
				l.Error().Err(err).Msg("failed to get seed from Veidemann")
				continue
			}
			sr := SeedRecord{
				SeedId:          seed.GetId(),
				Uri:             seed.GetMeta().GetName(),
				SeedDescription: seed.GetMeta().GetDescription(),
				EntityId:        seed.GetSeed().GetEntityRef().GetId(),
			}
			rec.Seeds = append(rec.Seeds, sr)

			entity, err := d.Client.GetConfigObject(context.Background(), seed.GetSeed().GetEntityRef())
			if err != nil {
				l.Warn().Err(err).Str("key", rec.Host).Str("entityId", sr.EntityId).Msg("failed to get entity from Veidemann")
				continue
			}
			sr.EntityName = entity.GetMeta().GetName()
			sr.EntityDescription = entity.GetMeta().GetDescription()
		}

		b, err := json.Marshal(rec)
		if err != nil {
			l.Error().Err(err).Msg("failed to marshal record to json")
			return
		}

		if _, err := w.Write(b); err != nil {
			l.Error().Err(err).Msg("failed to write record")
			return
		}
		if _, err := w.Write([]byte{'\n'}); err != nil {
			l.Error().Err(err).Msg("failed to write record")
			return
		}
	}
	err := d.Iterate(checkDuplicate)
	if err != nil {
		return err
	}

	log.Info().Int32("duplicate keys", nrOfDuplicateKeys).Int32("duplicate values", nrOfDuplicateValues).Msg("Duplicate report completed")
	return nil
}
