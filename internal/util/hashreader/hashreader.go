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

package hashreader

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
)

// HashReader is a wrapper around an io.Reader that calculates the SHA-256 hash of the read data.
type HashReader struct {
	reader io.Reader
	hasher hash.Hash
}

// NewReader creates a new HashReader.
func NewReader(r io.Reader) *HashReader {
	hasher := sha256.New()
	return &HashReader{
		reader: io.TeeReader(r, hasher),
		hasher: hasher,
	}
}

// Read reads from the underlying reader and updates the hash.
func (hr *HashReader) Read(p []byte) (int, error) {
	return hr.reader.Read(p)
}

// Verify returns true if the calculated hash matches the expected hash.
func (hr *HashReader) Verify(expected string) error {
	expectedHash, err := hex.DecodeString(expected)
	if err != nil {
		return err
	}

	if !hmac.Equal(hr.hasher.Sum(nil), expectedHash) {
		return errors.New("hash mismatch")
	}

	return nil
}
