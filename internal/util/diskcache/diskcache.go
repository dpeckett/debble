/*
 * Copyright 2024 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Immutos Community Edition License, Version 1.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *    http://immutos.com/licenses/LICENSE-1.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package diskcache

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/rogpeppe/go-internal/cache"
)

// DiskCache is a cache that stores http responses on disk.
type DiskCache struct {
	*cache.Cache
	namespace string
}

// NewDiskCache creates a new cache that stores responses in the given directory.
// The namespace is used to separate different caches in the same directory.
func NewDiskCache(dir, namespace string) (*DiskCache, error) {
	c, err := cache.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("error opening cache: %w", err)
	}

	c.Trim()

	return &DiskCache{
		Cache:     c,
		namespace: namespace,
	}, nil
}

func (c *DiskCache) Get(key string) ([]byte, bool) {
	responseBytes, _, err := c.Cache.GetBytes(c.getActionID(key))
	if err != nil {
		if !(errors.Is(err, os.ErrNotExist) || err.Error() == "cache entry not found") {
			slog.Warn("Error getting cached response",
				slog.String("key", key), slog.Any("error", err))
		} else {
			slog.Debug("Cache miss", slog.String("key", key))
		}

		return nil, false
	}

	slog.Debug("Cache hit", slog.String("key", key))

	return responseBytes, true
}

func (c *DiskCache) Set(key string, responseBytes []byte) {
	slog.Debug("Storing cached response", slog.String("key", key))

	if err := c.Cache.PutBytes(c.getActionID(key), responseBytes); err != nil {
		slog.Warn("Error setting cached response", slog.Any("error", err))
	}
}

func (c *DiskCache) Delete(key string) {}

func (c *DiskCache) getActionID(key string) cache.ActionID {
	h := cache.NewHash(c.namespace)
	_, _ = h.Write([]byte(key))
	return h.Sum()
}
