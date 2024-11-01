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

// Package exptypes from buildkit is not available in the Debian bookworm
// docker-dev package. Remove this as soon as the package is available.
package exptypes

import ocispecs "github.com/opencontainers/image-spec/specs-go/v1"

const (
	ExporterImageConfigKey = "containerimage.config"
	ExporterPlatformsKey   = "refs.platforms"
	OptKeyRewriteTimestamp = "rewrite-timestamp"
	OptKeySourceDateEpoch  = "source-date-epoch"
)

type Platforms struct {
	Platforms []Platform
}

type Platform struct {
	ID       string
	Platform ocispecs.Platform
}
