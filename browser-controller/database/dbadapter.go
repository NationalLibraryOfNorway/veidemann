/*
 * Copyright 2020 National Library of Norway.
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
	"fmt"
	"strings"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
)

type ConfigAdapter interface {
	GetConfigObject(context.Context, *configV1.ConfigRef) (*configV1.ConfigObject, error)
	GetScripts(context.Context, *configV1.BrowserConfig) ([]*configV1.ConfigObject, error)
}

type DbAdapter interface {
	GetConfigObject(context.Context, *configV1.ConfigRef) (*configV1.ConfigObject, error)
	GetConfigsForSelector(context.Context, configV1.Kind, *configV1.Label) ([]*configV1.ConfigObject, error)
}

type configAdapter struct {
	db DbAdapter
}

func NewConfigAdapter(db DbAdapter) ConfigAdapter {
	return &configAdapter{
		db: db,
	}
}

func (cc *configAdapter) GetConfigObject(ctx context.Context, ref *configV1.ConfigRef) (*configV1.ConfigObject, error) {
	return cc.db.GetConfigObject(ctx, ref)
}

// getConfigsForSelector fetches configObjects by selector string (key:value)
func (cc *configAdapter) getConfigsForSelector(ctx context.Context, selector string) ([]*configV1.ConfigObject, error) {
	t := strings.Split(selector, ":")
	label := &configV1.Label{
		Key:   t[0],
		Value: t[1],
	}

	return cc.db.GetConfigsForSelector(ctx, configV1.Kind_browserScript, label)
}

func (cc *configAdapter) GetScripts(ctx context.Context, browserConfig *configV1.BrowserConfig) ([]*configV1.ConfigObject, error) {
	var scripts []*configV1.ConfigObject
	for _, scriptRef := range browserConfig.ScriptRef {
		script, err := cc.GetConfigObject(ctx, scriptRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get script by reference %v: %w", scriptRef, err)
		}
		scripts = append(scripts, script)
	}
	for _, selector := range browserConfig.ScriptSelector {
		configs, err := cc.getConfigsForSelector(ctx, selector)
		if err != nil {
			return nil, fmt.Errorf("failed to get scripts by selector %s: %w", selector, err)
		}
		scripts = append(scripts, configs...)
	}
	return scripts, nil
}
