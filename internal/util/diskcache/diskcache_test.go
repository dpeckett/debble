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

package diskcache_test

import (
	"testing"

	"github.com/immutos/immutos/internal/testutil"
	"github.com/immutos/immutos/internal/util/diskcache"
	"github.com/stretchr/testify/require"
)

func TestDiskCache(t *testing.T) {
	testutil.SetupGlobals(t)

	cacheDir := t.TempDir()

	cache, err := diskcache.NewDiskCache(cacheDir, "test")
	require.NoError(t, err)

	t.Run("Exist", func(t *testing.T) {
		cache.Set("exist", []byte("data"))

		data, ok := cache.Get("exist")
		require.True(t, ok)
		require.Equal(t, []byte("data"), data)
	})

	t.Run("Non Exist", func(t *testing.T) {
		_, ok := cache.Get("non-exist")
		require.False(t, ok)
	})
}
