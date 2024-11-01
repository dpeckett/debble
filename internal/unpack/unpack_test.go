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

package unpack_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpeckett/archivefs/tarfs"
	"github.com/immutos/immutos/internal/testutil"
	"github.com/immutos/immutos/internal/unpack"
	"github.com/stretchr/testify/require"
)

func TestUnpack(t *testing.T) {
	testutil.SetupGlobals(t)

	tempDir := t.TempDir()

	ctx := context.Background()

	packagePaths := []string{
		filepath.Join(testutil.Root(), "testdata/debs/base-files_12.4+deb12u5_amd64.deb"),
		filepath.Join(testutil.Root(), "testdata/debs/base-passwd_3.6.1_amd64.deb"),
	}

	dpkgDatabaseArchivePath, dataArchivePaths, err := unpack.Unpack(ctx, tempDir, packagePaths)
	require.NoError(t, err)

	require.Len(t, dataArchivePaths, 2)
	require.Equal(t, "base-files_12.4+deb12u5_amd64_data.tar", filepath.Base(dataArchivePaths[0]))
	require.Equal(t, "base-passwd_3.6.1_amd64_data.tar", filepath.Base(dataArchivePaths[1]))

	dpkgDatabaseArchiveFile, err := os.Open(dpkgDatabaseArchivePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dpkgDatabaseArchiveFile.Close())
	})

	tarFS, err := tarfs.Open(dpkgDatabaseArchiveFile)
	require.NoError(t, err)

	var filesList []string
	err = fs.WalkDir(tarFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		filesList = append(filesList, path)
		return nil
	})
	require.NoError(t, err)

	expectedFilesList := []string{
		"var",
		"var/lib",
		"var/lib/dpkg",
		"var/lib/dpkg/info",
		"var/lib/dpkg/info/base-files.conffiles",
		"var/lib/dpkg/info/base-files.list",
		"var/lib/dpkg/info/base-files.md5sums",
		"var/lib/dpkg/info/base-files.postinst",
		"var/lib/dpkg/info/base-passwd.list",
		"var/lib/dpkg/info/base-passwd.md5sums",
		"var/lib/dpkg/info/base-passwd.postinst",
		"var/lib/dpkg/info/base-passwd.postrm",
		"var/lib/dpkg/info/base-passwd.preinst",
		"var/lib/dpkg/info/base-passwd.templates",
		"var/lib/dpkg/status",
	}

	require.ElementsMatch(t, expectedFilesList, filesList)
}
