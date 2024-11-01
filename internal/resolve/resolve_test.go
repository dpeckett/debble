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

package resolve_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dpeckett/deb822"
	"github.com/dpeckett/uncompr"
	"github.com/immutos/immutos/internal/database"
	"github.com/immutos/immutos/internal/resolve"
	"github.com/immutos/immutos/internal/testutil"
	"github.com/immutos/immutos/internal/types"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	testutil.SetupGlobals(t)

	f, err := os.Open(filepath.Join(testutil.Root(), "testdata/Packages.gz"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, f.Close())
	})

	dr, err := uncompr.NewReader(f)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dr.Close())
	})

	decoder, err := deb822.NewDecoder(dr, nil)
	require.NoError(t, err)

	var packageList []types.Package
	require.NoError(t, decoder.Decode(&packageList))

	packageDB := database.NewPackageDB()
	packageDB.AddAll(packageList)

	selectedDB, err := resolve.Resolve(packageDB, []string{"bash=5.2.15-2+b2"}, nil)
	require.NoError(t, err)

	var selectedNameVersions []string
	_ = selectedDB.ForEach(func(pkg types.Package) error {
		selectedNameVersions = append(selectedNameVersions,
			fmt.Sprintf("%s=%s", pkg.Name, pkg.Version))

		return nil
	})

	expectedNameVersions := []string{
		"base-files=12.4+deb12u5",
		"bash=5.2.15-2+b2",
		"debianutils=5.7-0.5~deb12u1",
		"gcc-12-base=12.2.0-14",
		"libc6=2.36-9+deb12u4",
		"libgcc-s1=12.2.0-14",
		"libtinfo6=6.4-4",
		"mawk=1.3.4.20200120-3.1",
	}

	require.ElementsMatch(t, expectedNameVersions, selectedNameVersions)
}
