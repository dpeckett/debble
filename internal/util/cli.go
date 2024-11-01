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
	"github.com/urfave/cli/v2"
)

// BeforeAll runs multiple BeforeFuncs in order
func BeforeAll(fns ...cli.BeforeFunc) cli.BeforeFunc {
	return func(c *cli.Context) error {
		for _, fn := range fns {
			if err := fn(c); err != nil {
				return err
			}
		}

		return nil
	}
}
