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

package testutil

import (
	"os"
	"path/filepath"
)

// Root finds the root directory of the Go module by looking for the go.mod file.
func Root() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Look for go.mod file by walking up the directory structure.
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		// Move to the parent directory.
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			panic("could not find root directory")
		}
		dir = parentDir
	}
}
