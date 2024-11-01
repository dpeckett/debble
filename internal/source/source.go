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
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/dpeckett/deb822"
	"github.com/dpeckett/deb822/types"
	"github.com/dpeckett/deb822/types/arch"
	"github.com/immutos/immutos/internal/keyring"
	latestrecipe "github.com/immutos/immutos/internal/recipe/v1alpha1"
)

const defaultDistribution = "stable"

var defaultComponents = []string{"main"}

// Source represents a Debian repository source.
type Source struct {
	keyring      openpgp.EntityList
	sourceURL    *url.URL
	distribution string
	components   []string
}

// NewSource creates a new Debian repository source.
func NewSource(ctx context.Context, conf latestrecipe.SourceConfig) (*Source, error) {
	distribution := defaultDistribution
	if conf.Distribution != "" {
		distribution = conf.Distribution
	}

	components := defaultComponents
	if len(conf.Components) > 0 {
		components = conf.Components
	}

	sourceURL, err := url.Parse(conf.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source URL: %w", err)
	}

	keyring, err := keyring.Load(ctx, conf.SignedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to read keyring: %w", err)
	}

	return &Source{
		keyring:      keyring,
		sourceURL:    sourceURL,
		distribution: distribution,
		components:   components,
	}, nil
}

// Components returns the components available in the source for the target architecture.
func (s *Source) Components(ctx context.Context, targetArch arch.Arch) ([]Component, error) {
	inReleaseURL, err := url.Parse(s.sourceURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse source URL: %w", err)
	}

	inReleaseURL.Path = path.Join(inReleaseURL.Path, "dists", s.distribution, "InRelease")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, inReleaseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download InRelease file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download InRelease file: %s", resp.Status)
	}

	decoder, err := deb822.NewDecoder(resp.Body, s.keyring)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if decoder.Signer() == nil {
		return nil, errors.New("InRelease file is not signed")
	}

	var release types.Release
	if err := decoder.Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to unmarshal InRelease file: %w", err)
	}

	allArch := arch.MustParse("all")
	var availableArchitectures []arch.Arch
	for _, releaseArch := range release.Architectures {
		if releaseArch.Is(&allArch) || releaseArch.Is(&targetArch) {
			availableArchitectures = append(availableArchitectures, releaseArch)
		}
	}

	if len(availableArchitectures) == 0 {
		slog.Warn("No architectures available")
		return nil, nil
	}

	desiredComponents := map[string]bool{}
	for _, component := range defaultComponents {
		desiredComponents[component] = true
	}
	for _, component := range s.components {
		desiredComponents[component] = true
	}

	var availableComponents []string
	for _, component := range release.Components {
		if desiredComponents[component] {
			availableComponents = append(availableComponents, component)
		}
	}

	if len(availableComponents) == 0 {
		slog.Warn("No components available")
		return nil, nil
	}

	var components []Component
	for _, component := range availableComponents {
		for _, arch := range availableArchitectures {
			componentURL, err := url.Parse(s.sourceURL.String())
			if err != nil {
				return nil, fmt.Errorf("failed to parse source URL: %w", err)
			}

			componentURL.Path = path.Join(componentURL.Path, "dists", s.distribution, component, "binary-"+arch.String())

			componentDir := path.Join(path.Base(component), "binary-"+arch.String())

			componentSHA256Sums := make(map[string]string)
			for _, hash := range release.SHA256 {
				if strings.HasPrefix(hash.Filename, componentDir) {
					componentSHA256Sums[strings.TrimPrefix(hash.Filename, componentDir+"/")] = hash.Hash
				}
			}

			components = append(components, Component{
				Name:       component,
				Arch:       arch,
				URL:        componentURL,
				SHA256Sums: componentSHA256Sums,
				keyring:    s.keyring,
				sourceURL:  s.sourceURL,
			})
		}
	}

	return components, nil
}
