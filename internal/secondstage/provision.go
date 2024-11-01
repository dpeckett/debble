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
	"context"
	"fmt"
	"log/slog"

	latestrecipe "github.com/immutos/immutos/internal/recipe/v1alpha1"
	"github.com/immutos/immutos/internal/secondstage/slimify"
	"github.com/immutos/immutos/internal/secondstage/users"
)

func Provision(ctx context.Context, rx *latestrecipe.Recipe) error {
	if rx.Options != nil && rx.Options.Slimify {
		slog.Info("Slimifying image")

		if err := slimify.Slimify(); err != nil {
			return fmt.Errorf("failed to slimify: %w", err)
		}
	}

	for _, groupConf := range rx.Groups {
		slog.Info("Creating or updating group", slog.String("name", groupConf.Name))

		group := users.Group{
			Name:    groupConf.Name,
			GID:     groupConf.GID,
			Members: groupConf.Members,
			System:  groupConf.System,
		}

		if err := users.CreateOrUpdateGroup(group); err != nil {
			return fmt.Errorf("failed to create group %q: %w", groupConf.Name, err)
		}
	}

	for _, userConf := range rx.Users {
		slog.Info("Creating or updating user", slog.String("name", userConf.Name))

		user := users.User{
			Name:     userConf.Name,
			UID:      userConf.UID,
			Groups:   userConf.Groups,
			HomeDir:  userConf.HomeDir,
			Shell:    userConf.Shell,
			Password: userConf.Password,
			System:   userConf.System,
		}

		if err := users.CreateOrUpdateUser(user); err != nil {
			return fmt.Errorf("failed to create or update user %q: %w", userConf.Name, err)
		}
	}

	return nil
}
