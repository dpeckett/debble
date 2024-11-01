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

package util

import (
	"log/slog"
	"strings"
)

// LevelFlag is a urfave/cli compatible flag for setting the log verbosity level.
type LevelFlag slog.Level

func FromSlogLevel(l slog.Level) *LevelFlag {
	f := LevelFlag(l)
	return &f
}

func (f *LevelFlag) Set(value string) error {
	return (*slog.Level)(f).UnmarshalText([]byte(strings.ToUpper(value)))
}

func (f *LevelFlag) String() string {
	return (*slog.Level)(f).String()
}
