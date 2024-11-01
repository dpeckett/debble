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

package resolve

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dpeckett/deb822/types/dependency"
	"github.com/dpeckett/deb822/types/version"
	"github.com/immutos/immutos/internal/database"
	"github.com/immutos/immutos/internal/types"
)

// Resolve resolves the dependencies of a list of packages, specified as a list
// of package name and optional version strings.
func Resolve(packageDB *database.PackageDB, includeNameVersions, excludeNameVersions []string) (*database.PackageDB, error) {
	requestedPackages := map[string]*version.Version{}
	candidateDB := database.NewPackageDB()

	// Parse excluded packages
	excludedPackages := map[string]*version.Version{}
	for _, excludeNameVersion := range excludeNameVersions {
		parts := strings.SplitN(excludeNameVersion, "=", 2)
		name := parts[0]

		var packageVersion *version.Version
		if len(parts) > 1 {
			v, err := version.Parse(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid excluded version: %s: %w", parts[1], err)
			}
			packageVersion = &v
		}
		excludedPackages[name] = packageVersion
	}

	for _, includeNameVersion := range includeNameVersions {
		parts := strings.SplitN(includeNameVersion, "=", 2)
		name := parts[0]

		var packageVersion *version.Version
		if len(parts) > 1 {
			v, err := version.Parse(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid version: %s: %w", parts[1], err)
			}

			packageVersion = &v
		}
		requestedPackages[name] = packageVersion

		if packageVersion != nil {
			pkg, exists := packageDB.ExactlyEqual(name, *packageVersion)
			if !exists {
				return nil, fmt.Errorf("unable to locate package: %s", includeNameVersion)
			}

			candidateDB.Add(*pkg)
		} else {
			packageList := packageDB.Get(name)
			if len(packageList) == 0 {
				return nil, fmt.Errorf("unable to locate package: %s", includeNameVersion)
			}

			candidateDB.AddAll(packageList)
		}
	}

	slog.Debug("Building dependency tree")

	var queue []types.Package
	_ = candidateDB.ForEach(func(pkg types.Package) error {
		queue = append(queue, pkg)
		return nil
	})

	visited := map[string]bool{}
	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]

		id := pkg.ID()
		if visited[id] {
			continue
		}
		visited[id] = true

		deps, err := getDependencies(packageDB, candidateDB, pkg)
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for package %s: %w", pkg.Name, err)
		}

		for _, depPkg := range deps {
			// Skip packages that are explicitly excluded.
			if _, excluded := excludedPackages[depPkg.Package.Name]; excluded {
				continue
			}

			if !visited[depPkg.ID()] {
				candidateDB.Add(depPkg)
				queue = append(queue, depPkg)
			}
		}
	}

	slog.Debug("Pruning candidates with unsatisfiable dependencies")

	pruneUnsatisfied(candidateDB, packageDB)

	// If there are multiple versions of the same package, select the newest
	// version.
	// TODO: shell out to a SAT solver to find the optimal solution.
	// TODO: handle conflicts etc.
	slog.Debug("Selecting newest version of each package")

	var selectedDB = database.NewPackageDB()
	_ = candidateDB.ForEach(func(pkg types.Package) error {
		// If the package is requested with an explicit version, only select it if the version matches.
		if packageVersion, ok := requestedPackages[pkg.Package.Name]; ok && packageVersion != nil {
			if pkg.Version.Compare(*packageVersion) == 0 {
				selectedDB.Add(pkg)
			}
			return nil
		}

		// If the package is already selected, only replace it if the new version
		// is higher.
		if existing := selectedDB.Get(pkg.Package.Name); len(existing) > 0 {
			if pkg.Version.Compare(existing[0].Version) > 0 {
				selectedDB.Remove(existing[0])
				selectedDB.Add(pkg)
			}
		} else {
			selectedDB.Add(pkg)
		}

		return nil
	})

	pruneUnsatisfied(selectedDB, packageDB)

	slog.Debug("Confirming requested packages are still selected")

	// Confirm all the requested packages are still selected.
	for name, version := range requestedPackages {
		if version != nil {
			if _, exists := selectedDB.ExactlyEqual(name, *version); !exists {
				return nil, fmt.Errorf("requested package %s=%s is not selected", name, version)
			}
		} else {
			if len(selectedDB.Get(name)) == 0 {
				return nil, fmt.Errorf("requested package %s is not selected", name)
			}
		}
	}

	return selectedDB, nil
}

// pruneUnsatisfied iteratively removes candidates with unsatisfiable dependencies.
func pruneUnsatisfied(candidateDB, packageDB *database.PackageDB) {
	for {
		var pruneList []types.Package
		_ = candidateDB.ForEach(func(pkg types.Package) error {
			if _, err := getDependencies(packageDB, candidateDB, pkg); err != nil {
				slog.Debug("Pruning unsatisfiable candidate",
					slog.String("name", pkg.Package.Name), slog.String("version", pkg.Version.String()),
					slog.Any("error", err))

				pruneList = append(pruneList, pkg)
			}

			return nil
		})

		for _, pkg := range pruneList {
			candidateDB.Remove(pkg)
		}

		if len(pruneList) == 0 {
			break
		}
	}
}

func getDependencies(packageDB, candidateDB *database.PackageDB, pkg types.Package) ([]types.Package, error) {
	var dependencies []types.Package

	var relations []dependency.Relation
	relations = append(relations, pkg.PreDepends.Relations...)
	relations = append(relations, pkg.Depends.Relations...)

	for _, rel := range relations {
		var resolved bool
		for _, possi := range rel.Possibilities {
			// TODO: implement all of the remainder of the debian relation constraints.

			var packageList []types.Package
			if possi.Version != nil {
				switch possi.Version.Operator {
				case "<<":
					packageList = packageDB.EarlierOrEqual(possi.Name, possi.Version.Version)
				case "<=":
					packageList = packageDB.EarlierOrEqual(possi.Name, possi.Version.Version)
				case "=":
					pkg, exists := packageDB.ExactlyEqual(possi.Name, possi.Version.Version)
					if exists {
						packageList = []types.Package{*pkg}
					}
				case ">=":
					packageList = packageDB.LaterOrEqual(possi.Name, possi.Version.Version)
				case ">>":
					packageList = packageDB.LaterOrEqual(possi.Name, possi.Version.Version)
				default:
					return nil, fmt.Errorf("unknown version relation operator: %s", possi.Version.Operator)
				}
			} else {
				packageList = packageDB.Get(possi.Name)
			}

			// Resolve virtual packages.
			var resolvedPackages []types.Package
			for _, pkg := range packageList {
				if pkg.IsVirtual {
					if resolvedPkg, err := resolveVirtualPackage(packageDB, candidateDB, pkg); err == nil {
						resolvedPackages = append(resolvedPackages, resolvedPkg)
					} else {
						slog.Debug("Failed to resolve virtual package",
							slog.String("name", pkg.Package.Name), slog.String("version", pkg.Version.String()),
							slog.Any("error", err))
					}
				} else {
					resolvedPackages = append(resolvedPackages, pkg)
				}
			}

			if len(resolvedPackages) > 0 {
				dependencies = append(dependencies, resolvedPackages...)
				resolved = true
				break
			}
		}

		if !resolved {
			return nil, fmt.Errorf("unsatisfiable dependency: %s", rel.String())
		}
	}

	return dependencies, nil
}

func resolveVirtualPackage(packageDB, candidateDB *database.PackageDB, virtualPkg types.Package) (types.Package, error) {
	var virtualProviders []types.Package
	for _, provider := range virtualPkg.Providers {
		if pkg, exists := packageDB.ExactlyEqual(provider.Package.Name, provider.Version); exists {
			virtualProviders = append(virtualProviders, *pkg)
		}
	}

	if len(virtualProviders) == 0 {
		return types.Package{}, fmt.Errorf("unsatisfiable dependency: %s", virtualPkg.Name)
	} else if len(virtualProviders) == 1 {
		return virtualProviders[0], nil
	} else {
		// Has a provider already been selected? Eg. its part of the candidate list.
		for _, pkg := range virtualProviders {
			if _, exists := candidateDB.ExactlyEqual(pkg.Package.Name, pkg.Version); exists {
				return pkg, nil
			}
		}

		// Is one of the providers marked as required priority?
		for _, pkg := range virtualProviders {
			if pkg.Priority == "required" {
				return pkg, nil
			}
		}

		return types.Package{}, fmt.Errorf("virtual package with multiple installation candidates: %s", virtualPkg.Name)
	}
}
