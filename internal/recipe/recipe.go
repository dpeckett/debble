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

package recipe

import (
	"fmt"
	"io"

	recipetypes "github.com/immutos/immutos/internal/recipe/types"
	latestrecipe "github.com/immutos/immutos/internal/recipe/v1alpha1"
	"gopkg.in/yaml.v3"
)

// FromYAML reads the given reader and returns a recipe object.
func FromYAML(r io.Reader) (*latestrecipe.Recipe, error) {
	recipeBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipe from reader: %w", err)
	}

	var typeMeta recipetypes.TypeMeta
	if err := yaml.Unmarshal(recipeBytes, &typeMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal type meta from recipe file: %w", err)
	}

	var versionedRecipe recipetypes.Typed
	switch typeMeta.APIVersion {
	case latestrecipe.APIVersion:
		versionedRecipe, err = latestrecipe.GetByKind(typeMeta.Kind)
	default:
		return nil, fmt.Errorf("unsupported api version: %s", typeMeta.APIVersion)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe by kind %q: %w", typeMeta.Kind, err)
	}

	if err := yaml.Unmarshal(recipeBytes, versionedRecipe); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recipe from recipe file: %w", err)
	}

	versionedRecipe, err = MigrateToLatest(versionedRecipe)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate recipe: %w", err)
	}

	return versionedRecipe.(*latestrecipe.Recipe), nil
}

// ToYAML writes the given recipe object to the given writer.
func ToYAML(w io.Writer, versionedRecipe recipetypes.Typed) error {
	versionedRecipe.PopulateTypeMeta()

	if err := yaml.NewEncoder(w).Encode(versionedRecipe); err != nil {
		return fmt.Errorf("failed to marshal recipe: %w", err)
	}

	return nil
}

// MigrateToLatest migrates the given recipe object to the latest version.
func MigrateToLatest(versionedRecipe recipetypes.Typed) (recipetypes.Typed, error) {
	switch recipe := versionedRecipe.(type) {
	case *latestrecipe.Recipe:
		// Nothing to do, already at the latest version.
		return recipe, nil
	default:
		return nil, fmt.Errorf("unsupported recipe version: %s", recipe.GetAPIVersion())
	}
}
