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

package database_test

import (
	"testing"

	debtypes "github.com/dpeckett/deb822/types"
	"github.com/dpeckett/deb822/types/dependency"
	"github.com/dpeckett/deb822/types/version"
	"github.com/immutos/immutos/internal/database"
	"github.com/immutos/immutos/internal/testutil"
	"github.com/immutos/immutos/internal/types"
	"github.com/stretchr/testify/require"
)

func TestPackageDB(t *testing.T) {
	testutil.SetupGlobals(t)

	db := database.NewPackageDB()

	db.AddAll([]types.Package{
		{
			Package: debtypes.Package{
				Name:    "foo",
				Version: version.MustParse("1.0"),
			},
		},
		{
			Package: debtypes.Package{
				Name:    "foo",
				Version: version.MustParse("1.1"),
			},
		},
		{
			Package: debtypes.Package{
				Name:    "bar",
				Version: version.MustParse("2.0"),
			},
		},
	})

	require.Equal(t, 3, db.Len())

	t.Run("Get", func(t *testing.T) {
		t.Run("All", func(t *testing.T) {
			packages := db.Get("foo")

			require.Len(t, packages, 2)
		})

		t.Run("Strictly Earlier", func(t *testing.T) {
			packages := db.StrictlyEarlier("foo", version.MustParse("1.1"))

			require.Len(t, packages, 1)
			require.Equal(t, "foo", packages[0].Name)
			require.Equal(t, version.MustParse("1.0"), packages[0].Version)
		})

		t.Run("Earlier or Equal", func(t *testing.T) {
			packages := db.EarlierOrEqual("foo", version.MustParse("1.1"))

			require.Len(t, packages, 2)
		})

		t.Run("Exact Version", func(t *testing.T) {
			pkg, exists := db.ExactlyEqual("foo", version.MustParse("1.0"))

			require.True(t, exists)
			require.Equal(t, "foo", pkg.Name)
			require.Equal(t, version.MustParse("1.0"), pkg.Version)
		})

		t.Run("Exact Version (Missing)", func(t *testing.T) {
			_, exists := db.ExactlyEqual("foo", version.MustParse("1.2"))

			require.False(t, exists)
		})

		t.Run("Later or Equal", func(t *testing.T) {
			packages := db.LaterOrEqual("foo", version.MustParse("1.0"))

			require.Len(t, packages, 2)
			require.Equal(t, "foo", packages[0].Name)
			require.Equal(t, version.MustParse("1.0"), packages[0].Version)
			require.Equal(t, version.MustParse("1.1"), packages[1].Version)
		})

		t.Run("Strictly Later", func(t *testing.T) {
			packages := db.StrictlyLater("foo", version.MustParse("1.0"))

			require.Len(t, packages, 1)
			require.Equal(t, "foo", packages[0].Name)
			require.Equal(t, version.MustParse("1.1"), packages[0].Version)
		})
	})

	t.Run("Add and Remove", func(t *testing.T) {
		pkg := types.Package{
			Package: debtypes.Package{
				Name:    "baz",
				Version: version.MustParse("3.0"),
			},
		}

		db.Add(pkg)

		require.Equal(t, 4, db.Len())

		db.Remove(pkg)

		require.Equal(t, 3, db.Len())
	})

	t.Run("Virtual Packages", func(t *testing.T) {
		pkg := types.Package{
			Package: debtypes.Package{
				Name:    "baz",
				Version: version.MustParse("3.0"),
				Provides: dependency.Dependency{
					Relations: []dependency.Relation{
						{
							Possibilities: []dependency.Possibility{{Name: "bazz"}},
						},
					},
				},
			},
		}

		db.Add(pkg)

		packages := db.Get("bazz")

		require.Len(t, packages, 1)
		require.Equal(t, "bazz", packages[0].Name)
		require.True(t, packages[0].IsVirtual)
		require.Equal(t, "baz", packages[0].Providers[0].Name)
		require.Equal(t, version.MustParse("3.0"), packages[0].Providers[0].Version)
	})
}
