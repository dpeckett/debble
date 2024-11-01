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

package v1alpha1

import (
	"fmt"

	"github.com/immutos/immutos/internal/recipe/types"
)

const APIVersion = "com.immutos/v1alpha1"

type Recipe struct {
	types.TypeMeta `yaml:",inline"`
	// Options contains configuration options for the image.
	Options *OptionsConfig `yaml:"options,omitempty"`
	// Sources is a list of apt repositories to use for downloading packages.
	Sources []SourceConfig `yaml:"sources"`
	// Packages is the package configuration.
	Packages PackagesConfig `yaml:"packages"`
	// Groups is a list of groups to create.
	Groups []GroupConfig `yaml:"groups,omitempty"`
	// Users is a list of users to create.
	Users []UserConfig `yaml:"users,omitempty"`
	// Container is the OCI image configuration.
	Container *ContainerConfig `yaml:"container,omitempty"`
}

// OptionsConfig contains configuration options for the image.
type OptionsConfig struct {
	// OmitRequired specifies whether to omit priority required packages from the installation.
	// By default, any packages marked as priority required will be installed.
	OmitRequired bool `yaml:"omitRequired,omitempty"`
	// Slimify specifies whether to slimify the image by removing unnecessary files.
	Slimify bool `yaml:"slimify,omitempty"`
	// DownloadOnly specifies whether to only download packages and not install them.
	DownloadOnly bool `yaml:"downloadOnly,omitempty"`
}

// SourceConfig is the configuration for an apt repository.
type SourceConfig struct {
	// URL is the URL of the repository.
	URL string `yaml:"url"`
	// Signed by is a public key URL (https) or file path to use for verifying the repository.
	SignedBy string `yaml:"signedBy"`
	// Distribution specifies the Debian distribution name (e.g., bullseye, buster)
	// or class (e.g., stable, testing). If not specified, defaults to "stable".
	Distribution string `yaml:"distribution,omitempty"`
	// Components is a list of components to use from the repository.
	// If not specified, defaults to ["main"].
	Components []string `yaml:"components,omitempty"`
}

// PackagesConfig is the configuration for packages.
type PackagesConfig struct {
	// Include is a list of packages to install.
	Include []string `yaml:"include,omitempty"`
	// Exclude is a list of packages to exclude from installation.
	Exclude []string `yaml:"exclude,omitempty"`
}

// GroupConfig is the configuration for a group.
type GroupConfig struct {
	// Name is the name of the group.
	Name string `yaml:"name"`
	// GID is the group ID to use for the group.
	GID *uint `yaml:"gid,omitempty"`
	// Members is a list of users to add to the group.
	Members []string `yaml:"members,omitempty"`
	// System specifies whether the group is a system group.
	System bool `yaml:"system,omitempty"`
}

// UserConfig is the configuration for a user.
type UserConfig struct {
	// Name is the name of the user.
	Name string `yaml:"name"`
	// UID is the user ID to use for the user.
	UID *uint `yaml:"uid,omitempty"`
	// Groups is a list of groups to add the user to.
	// The first group in the list will be treated as the users primary group.
	Groups []string `yaml:"groups,omitempty"`
	// HomeDir is the home directory for the user, if not specified, defaults
	// to /home/<name>.
	HomeDir string `yaml:"homeDir,omitempty"`
	// Shell is the shell for the user, if not specified, defaults to
	// /usr/sbin/nologin.
	Shell string `yaml:"shell,omitempty"`
	// Password is the optional password for the user.
	// If not specified, password authentication will be disabled.
	Password string `yaml:"password,omitempty"`
	// System specifies whether the user is a system user.
	System bool `yaml:"system,omitempty"`
}

// ContainerConfig is the configuration for the container.
type ContainerConfig struct {
	// User defines the username or UID which the process in the container should run as.
	User string `yaml:"user,omitempty"`
	// ExposedPorts a set of ports to expose from a container running this image.
	ExposedPorts map[string]struct{} `yaml:"exposedPorts,omitempty"`
	// Env is a list of additional environment variables to be used in a container.
	Env []string `yaml:"env,omitempty"`
	// Entrypoint defines a list of arguments to use as the command to execute when
	// the container starts.
	Entrypoint []string `yaml:"entrypoint,omitempty"`
	// Cmd defines the default arguments to the entrypoint of the container.
	Cmd []string `yaml:"cmd,omitempty"`
	// Volumes is a set of directories describing where the process is likely write
	// data specific to a container instance.
	Volumes map[string]struct{} `yaml:"volumes,omitempty"`
	// WorkingDir sets the current working directory of the entrypoint process in the container.
	WorkingDir string `yaml:"workingDir,omitempty"`
	// Labels contains arbitrary metadata for the container.
	Labels map[string]string `yaml:"labels,omitempty"`
	// StopSignal contains the system call signal that will be sent to the container to exit.
	StopSignal string `yaml:"stopSignal,omitempty"`
}

func (r *Recipe) GetAPIVersion() string {
	return APIVersion
}

func (r *Recipe) GetKind() string {
	return "Recipe"
}

func (r *Recipe) PopulateTypeMeta() {
	r.TypeMeta = types.TypeMeta{
		APIVersion: APIVersion,
		Kind:       "Recipe",
	}
}

func GetByKind(kind string) (types.Typed, error) {
	switch kind {
	case "Recipe":
		return &Recipe{}, nil
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}
