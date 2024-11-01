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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/immutos/immutos/internal/util"
	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

func TestCreateOrUpdateGroup(t *testing.T) {
	dir := t.TempDir()

	// Set the file paths to the temp directory.
	groupFilePath = filepath.Join(dir, "group")
	groupShadowFilePath = filepath.Join(dir, "gshadow")
	passwdFilePath = filepath.Join(dir, "passwd")
	shadowFilePath = filepath.Join(dir, "shadow")

	// copy the test files to the temp directory.
	require.NoError(t, cp.Copy("testdata/reference", dir))

	// Create a new group.
	require.NoError(t, CreateOrUpdateGroup(Group{
		Name:    "testgroup",
		Members: []string{"user1", "user2"},
	}))

	// Update an existing group.
	require.NoError(t, CreateOrUpdateGroup(Group{
		Name:    "sudo",
		GID:     util.PointerTo(uint(27)),
		Members: []string{"user1", "user2"},
		System:  true,
	}))

	// Test invalid group name.
	require.Error(t, CreateOrUpdateGroup(Group{Name: "test:group"}))
	require.Error(t, CreateOrUpdateGroup(Group{Name: strings.Repeat("a", 33)}))

	// Confirm the group file contents.
	require.FileExists(t, groupFilePath)
	require.FileExists(t, groupFilePath+"-")

	buf, err := os.ReadFile(groupFilePath)
	require.NoError(t, err)

	expectedGroupContents, err := os.ReadFile("testdata/group_test/group")
	require.NoError(t, err)

	require.Equal(t, string(expectedGroupContents), string(buf))

	// Confirm the group shadow file contents.
	require.FileExists(t, groupShadowFilePath)
	require.FileExists(t, groupShadowFilePath+"-")

	buf, err = os.ReadFile(groupShadowFilePath)
	require.NoError(t, err)

	expectedGroupShadowContents, err := os.ReadFile("testdata/group_test/gshadow")
	require.NoError(t, err)

	require.Equal(t, string(expectedGroupShadowContents), string(buf))
}
