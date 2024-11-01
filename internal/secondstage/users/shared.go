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

package users

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"

	cp "github.com/otiai10/copy"
)

var validNameRegexp = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]{0,31}$`)

func updateFile(path string, perm fs.FileMode, updateFunc func(*lineReader) (string, error)) error {
	if err := cp.Copy(path, path+"-", cp.Options{Sync: true}); err != nil {
		return fmt.Errorf("failed to backup %q: %w", path, err)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	contents, err := updateFunc(&lineReader{bufio.NewReader(f)})
	if err != nil {
		return err
	}

	if err := f.Truncate(0); err != nil {
		return err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	_, err = f.WriteString(contents)
	return err
}

type lineReader struct {
	*bufio.Reader
}

func (r *lineReader) nextLine() (string, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return "", err
		}

		if len(line) == 0 {
			return "", io.EOF
		}
	}

	return strings.TrimSpace(string(line)), nil
}
