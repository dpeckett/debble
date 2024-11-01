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

package keyring

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// Load reads an OpenPGP keyring from a file or URL.
func Load(ctx context.Context, key string) (openpgp.EntityList, error) {
	if len(key) == 0 {
		return openpgp.EntityList{}, nil
	}

	// If the key is a URL, download it.
	if strings.Contains(key, "://") {
		slog.Debug("Downloading key", slog.String("url", key))

		keyURL, err := url.Parse(key)
		if err != nil {
			return nil, err
		}

		if keyURL.Scheme != "https" {
			return nil, errors.New("key URL must be HTTPS")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, keyURL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to download key: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download key: %s", resp.Status)
		}

		// ReadArmoredKeyRing() doesn't read the entire response body, so we need
		// to do it ourselves (so that response caching will work as expected).
		keyringData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return openpgp.ReadArmoredKeyRing(bytes.NewReader(keyringData))
	} else { // If the key is a file, open it.
		slog.Debug("Reading key file", slog.String("path", key))

		f, err := os.Open(key)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		return openpgp.ReadArmoredKeyRing(f)
	}
}
