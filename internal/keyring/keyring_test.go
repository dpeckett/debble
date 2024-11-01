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

package keyring_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/immutos/immutos/internal/keyring"
	"github.com/immutos/immutos/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestKeyringRead(t *testing.T) {
	testutil.SetupGlobals(t)

	ctx := context.Background()

	t.Run("Web", func(t *testing.T) {
		keyring, err := keyring.Load(ctx, "https://ftp-master.debian.org/keys/archive-key-12.asc")
		require.NoError(t, err)

		require.NotEmpty(t, keyring)
	})

	t.Run("File", func(t *testing.T) {
		keyring, err := keyring.Load(ctx, filepath.Join(testutil.Root(), "testdata/archive-key-12.asc"))
		require.NoError(t, err)

		require.NotEmpty(t, keyring)
	})
}
