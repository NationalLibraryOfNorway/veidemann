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

package database

import (
	"context"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
)

type ContentAdapter interface {
	HasCrawledContent(ctx context.Context, collection, key string) (*contentwriterV1.CrawledContent, error)
	WriteCrawledContent(ctx context.Context, collection string, ttl time.Duration, crawledContent *contentwriterV1.CrawledContent) error
}

type ConfigAdapter interface {
	GetConfigObject(context.Context, *configV1.ConfigRef) (*configV1.ConfigObject, error)
}

type configCache struct {
	db    ConfigAdapter
	cache *cache
}

func NewConfigCache(db ConfigAdapter, ttl time.Duration) ConfigAdapter {
	return &configCache{
		db:    db,
		cache: newCache(ttl),
	}
}

func (cc *configCache) GetConfigObject(ctx context.Context, ref *configV1.ConfigRef) (*configV1.ConfigObject, error) {
	cached := cc.cache.Get(ref.Id)
	if cached != nil {
		return cached, nil
	}

	result, err := cc.db.GetConfigObject(ctx, ref)
	if err != nil {
		return nil, err
	}

	cc.cache.Set(result.Id, result)

	return result, nil
}
