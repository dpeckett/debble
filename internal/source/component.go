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

package source

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/dpeckett/deb822"
	"github.com/dpeckett/deb822/types/arch"
	"github.com/dpeckett/uncompr"
	"github.com/immutos/immutos/internal/types"
	"github.com/immutos/immutos/internal/util/hashreader"
)

// Component represents a component of a Debian repository.
type Component struct {
	// Name is the name of the component.
	Name string
	// Arch is the architecture of the component.
	Arch arch.Arch
	// URL is the base URL of the component.
	URL *url.URL
	// SHA256Sums are the SHA256 sums of files in the component.
	SHA256Sums map[string]string
	// Internal fields.
	keyring   openpgp.EntityList
	sourceURL *url.URL
}

func (c *Component) Packages(ctx context.Context) ([]types.Package, time.Time, error) {
	var errs error

	for _, name := range []string{"Packages.xz", "Packages.gz", "Packages"} {
		packagesURL, err := url.Parse(c.URL.String())
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to parse component URL: %w", err)
		}

		packagesURL.Path = path.Join(packagesURL.Path, name)

		slog.Debug("Attempting to download Packages file", slog.String("url", packagesURL.String()))

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, packagesURL.String(), nil)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to download %s file: %w", name, err))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errs = errors.Join(errs, fmt.Errorf("failed to download %s file: %s", name, resp.Status))
			continue
		}

		// Get the last updated time.
		lastUpdated, err := http.ParseTime(resp.Header.Get("Last-Modified"))
		if err != nil {
			slog.Warn("Failed to parse Last-Modified header",
				slog.String("url", packagesURL.String()), slog.Any("error", err))
		}

		hr := hashreader.NewReader(resp.Body)

		dr, err := uncompr.NewReader(hr)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to decompress %s file: %w", name, err))
			continue
		}
		defer dr.Close()

		slog.Debug("Unmarshalling Packages file", slog.String("url", packagesURL.String()))

		decoder, err := deb822.NewDecoder(dr, c.keyring)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to create decoder: %w", err))
			continue
		}

		var packageList []types.Package
		if err := decoder.Decode(&packageList); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to unmarshal %s file: %w", name, err))
			continue
		}

		if err := hr.Verify(c.SHA256Sums[name]); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to verify %s file: %w", name, err))
			continue
		}

		packageURL, err := url.Parse(c.sourceURL.String())
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to parse source URL: %w", err)
		}
		basePath := packageURL.Path

		for i := range packageList {
			packageURL.Path = path.Join(basePath, packageList[i].Filename)
			packageList[i].URLs = append(packageList[i].URLs, packageURL.String())
		}

		return packageList, lastUpdated, nil
	}

	return nil, time.Time{}, fmt.Errorf("failed to download Packages file: %w", errs)
}
