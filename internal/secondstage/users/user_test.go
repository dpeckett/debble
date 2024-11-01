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

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

func TestCreateOrUpdateUser(t *testing.T) {
	dir := t.TempDir()

	// Set the file paths to the temp directory.
	groupFilePath = filepath.Join(dir, "group")
	groupShadowFilePath = filepath.Join(dir, "gshadow")
	passwdFilePath = filepath.Join(dir, "passwd")
	shadowFilePath = filepath.Join(dir, "shadow")

	// copy the test files to the temp directory.
	require.NoError(t, cp.Copy("testdata/reference", dir))

	// Create a group for the user.
	require.NoError(t, CreateOrUpdateGroup(Group{Name: "testgroup"}))

	// Create a new user.
	err := CreateOrUpdateUser(User{
		Name:     "testuser",
		Groups:   []string{"testgroup", "sudo"},
		HomeDir:  "/home/testuser",
		Shell:    "/bin/bash",
		Password: "testpassword",
	})
	require.NoError(t, err)

	require.FileExists(t, passwdFilePath)
	require.FileExists(t, passwdFilePath+"-")

	buf, err := os.ReadFile(passwdFilePath)
	require.NoError(t, err)

	expectedPasswdContents, err := os.ReadFile("testdata/user_test/passwd")
	require.NoError(t, err)

	require.Equal(t, string(expectedPasswdContents), string(buf))

	require.FileExists(t, shadowFilePath)
	require.FileExists(t, shadowFilePath+"-")

	buf, err = os.ReadFile(shadowFilePath)
	require.NoError(t, err)

	// Mask out the bcrypt hash.
	start := strings.Index(string(buf), "$2a$10") + 6
	end := strings.Index(string(buf[start:]), ":")

	buf = []byte(string(buf[:start]) + string(buf[start+end:]))

	expectedShadowContents, err := os.ReadFile("testdata/user_test/shadow")
	require.NoError(t, err)

	require.Equal(t, string(expectedShadowContents), string(buf))

	require.FileExists(t, groupFilePath)
	require.FileExists(t, groupFilePath+"-")

	buf, err = os.ReadFile(groupFilePath)
	require.NoError(t, err)

	expectedGroupContents, err := os.ReadFile("testdata/user_test/group")
	require.NoError(t, err)

	require.Equal(t, string(expectedGroupContents), string(buf))
}
