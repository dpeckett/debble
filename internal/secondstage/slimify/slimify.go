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

package slimify

import (
	"bytes"
	_ "embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/moby/patternmatcher"
)

//go:embed .slimify
var dotSlimify []byte

var excludedDirs = map[string]bool{
	"/dev":  true,
	"/proc": true,
	"/sys":  true,
	"/tmp":  true,
}

// Slimify the image by removing unnecessary files.
func Slimify() error {
	patterns, err := dockerignore.ReadAll(bytes.NewReader(dotSlimify))
	if err != nil {
		return fmt.Errorf("failed to read patterns: %w", err)
	}

	pm, err := patternmatcher.New(patterns)
	if err != nil {
		return fmt.Errorf("failed to create pattern matcher: %w", err)
	}

	// First walk the root filesystem and collect paths to remove.
	var pathsToRemove []string
	err = filepath.WalkDir("/", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				slog.Warn("Skipping", "path", path, "error", err)
				return nil
			}

			return err
		}

		// Skip special directories.
		if excludedDirs[path] {
			return fs.SkipDir
		}

		matches, err := pm.MatchesOrParentMatches(strings.TrimPrefix(path, "/"))
		if err != nil {
			return fmt.Errorf("failed to match %s: %w", path, err)
		}

		if matches {
			pathsToRemove = append(pathsToRemove, path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk root filesystem: %w", err)
	}

	// Sort the paths in reverse order so that we remove files before directories.
	sort.Slice(pathsToRemove, func(i, j int) bool {
		return len(pathsToRemove[i]) > len(pathsToRemove[j])
	})

	// Remove the paths.
	for _, path := range pathsToRemove {
		fi, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if fi.IsDir() {
			empty, err := isDirEmpty(path)
			if err != nil {
				return fmt.Errorf("failed to check if %s is empty: %w", path, err)
			}

			if !empty {
				continue
			}
		}

		slog.Debug("Removing", slog.String("path", path))

		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

func isDirEmpty(path string) (bool, error) {
	var filenames []string
	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			filenames = append(filenames, path)
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return len(filenames) == 0, nil
}
