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

package types

import (
	debtypes "github.com/dpeckett/deb822/types"
	"github.com/google/btree"
)

// Package represents a Debian package.
type Package struct {
	debtypes.Package
	// Additional fields that are not part of the standard control file but are
	// used internally by immutos.

	// URLs is a list of URLs that the package can be downloaded from.
	URLs []string `json:"-"`
	// IsVirtual is true if the package is a virtual package.
	IsVirtual bool `json:"-"`
	// Providers lists packages that provide this virtual package.
	Providers []Package `json:"-"`
}

func (p Package) Compare(other Package) int {
	return p.Package.Compare(other.Package)
}

func (p Package) Less(than btree.Item) bool {
	return p.Package.Compare(than.(Package).Package) < 0
}
