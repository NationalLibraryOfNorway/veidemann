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

	"github.com/redis/go-redis/v9"
)

func loadRedisScript(ctx context.Context, client *redis.Client, scriptSrc string) (*redis.Script, error) {
	// create script
	script := redis.NewScript(scriptSrc)

	// load script if it doesn't exist in redis
	boolSlice, err := script.Exists(ctx, client).Result()
	if err != nil {
		return nil, err
	}
	for _, exists := range boolSlice {
		if !exists {
			if err := script.Load(ctx, client).Err(); err != nil {
				return nil, err
			}
		}
	}

	return script, nil
}
