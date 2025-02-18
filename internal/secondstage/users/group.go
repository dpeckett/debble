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
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/immutos/immutos/internal/util"
)

const (
	systemGIDMin uint = 100
	systemGIDMax uint = 999
	userGIDMin   uint = 1000
	userGIDMax   uint = 60000
)

var (
	// For testing.
	groupFilePath       = "/etc/group"
	groupShadowFilePath = "/etc/gshadow"
)

type Group struct {
	Name    string
	GID     *uint
	System  bool
	Members []string
}

// CreateOrUpdateGroup creates or updates a group.
func CreateOrUpdateGroup(group Group) error {
	if !validNameRegexp.MatchString(group.Name) {
		return fmt.Errorf("invalid group name %q", group.Name)
	}

	if group.GID == nil {
		var err error
		group.GID, err = getNextFreeGID(group.System)
		if err != nil {
			return err
		}
	}

	if err := updateGroupFile(group); err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	if err := updateGroupShadowFile(group); err != nil {
		return fmt.Errorf("failed to update gshadow: %w", err)
	}

	return nil
}

func getNextFreeGID(system bool) (*uint, error) {
	groups, err := loadGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to parse group file: %w", err)
	}

	minGID := userGIDMin
	if system {
		minGID = systemGIDMin
	}

	for gid := minGID; gid <= userGIDMax; gid++ {
		if _, exists := groups[gid]; !exists {
			return &gid, nil
		}
	}

	return nil, errors.New("no available GID")
}

func loadGroups() (map[uint]Group, error) {
	f, err := os.Open(groupFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	groups := make(map[uint]Group)

	lr := &lineReader{bufio.NewReader(f)}
	for {
		line, err := lr.nextLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, err
		}

		// Skip comments.
		if line[0] == '#' {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			return nil, fmt.Errorf("invalid group entry: %q", line)
		}

		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid GID: %w", err)
		}

		var members []string
		if len(fields[3]) > 0 {
			members = strings.Split(fields[3], ",")
		}

		groups[uint(gid)] = Group{
			Name:    fields[0],
			GID:     util.PointerTo(uint(gid)),
			System:  uint(gid) < userGIDMin,
			Members: members,
		}
	}

	return groups, nil
}

func updateGroupFile(group Group) error {
	updateFunc := func(lr *lineReader) (string, error) {
		updatedEntry := fmt.Sprintf("%s:x:%d:%s", group.Name, *group.GID, strings.Join(deduplicate(group.Members), ","))
		found := false

		var sb strings.Builder
		for {
			line, err := lr.nextLine()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				return "", err
			}

			if strings.HasPrefix(line, group.Name+":") {
				sb.WriteString(updatedEntry)
				found = true
			} else {
				sb.WriteString(line)
			}
			sb.WriteRune('\n')
		}
		if !found {
			sb.WriteString(updatedEntry)
			sb.WriteRune('\n')
		}

		return sb.String(), nil
	}

	return updateFile(groupFilePath, 0o644, updateFunc)
}

func updateGroupShadowFile(group Group) error {
	updateFunc := func(lr *lineReader) (string, error) {
		updatedEntry := fmt.Sprintf("%s:!::%s", group.Name, strings.Join(deduplicate(group.Members), ","))
		found := false

		var sb strings.Builder
		for {
			line, err := lr.nextLine()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				return "", err
			}

			if strings.HasPrefix(string(line), group.Name+":") {
				sb.WriteString(updatedEntry)
				found = true
			} else {
				sb.WriteString(line)
			}
			sb.WriteRune('\n')
		}
		if !found {
			sb.WriteString(updatedEntry)
			sb.WriteRune('\n')
		}

		return sb.String(), nil
	}

	// Do we have a gshadow file?
	if _, err := os.Stat(groupShadowFilePath); os.IsNotExist(err) {
		slog.Warn("No gshadow file found, skipping")

		return nil
	}

	return updateFile(groupShadowFilePath, 0o400, updateFunc)
}

func deduplicate(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, str := range slice {
		if _, ok := seen[str]; !ok {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}
