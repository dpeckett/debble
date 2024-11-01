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

package secondstage

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

// usrMergeDirectories is the complete list of directories that can be merged into /usr.
var usrMergeDirectories = []string{"/bin", "/lib", "/lib32", "/lib64", "/libo32", "/libx32", "/sbin"}

// MergeUsr merges the /usr directory into the root filesystem
// See: https://wiki.debian.org/UsrMerge
func MergeUsr() error {
	for _, dir := range usrMergeDirectories {
		// The architecture does not have this directory.
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// The directory is already usr merged.
		if info, err := os.Lstat(dir); err == nil && info.Mode()&os.ModeSymlink != 0 {
			slog.Info("Directory is already usr merged", slog.String("dir", dir))
			continue
		}

		canonDir := filepath.Join("/usr", dir)

		slog.Info("Merging into /usr", slog.String("dir", dir), slog.String("canonDir", canonDir))

		if err := cp.Copy(dir, canonDir, cp.Options{OnSymlink: func(src string) cp.SymlinkAction {
			return cp.Shallow
		}}); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", dir, canonDir, err)
		}

		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to remove %s: %w", dir, err)
		}

		if err := os.Symlink(canonDir, dir); err != nil {
			return fmt.Errorf("failed to symlink %s -> /usr%s: %w", dir, canonDir, err)
		}
	}

	return nil
}
