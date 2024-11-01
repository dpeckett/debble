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

package hashreader_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/immutos/immutos/internal/testutil"
	"github.com/immutos/immutos/internal/util/hashreader"
	"github.com/stretchr/testify/require"
)

func TestHashReader(t *testing.T) {
	testutil.SetupGlobals(t)

	data := []byte("The quick brown fox jumps over the lazy dog")

	// Create a HashReader
	reader := bytes.NewReader(data)
	hashReader := hashreader.NewReader(reader)

	// Read the data
	readData, err := io.ReadAll(hashReader)
	require.NoError(t, err)
	require.Equal(t, data, readData)

	// Verify the checksum
	expected := "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"
	require.NoError(t, hashReader.Verify(expected))
}
