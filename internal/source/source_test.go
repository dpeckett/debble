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

package source_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/dpeckett/deb822/types/arch"
	latestrecipe "github.com/immutos/immutos/internal/recipe/v1alpha1"
	"github.com/immutos/immutos/internal/source"
	"github.com/immutos/immutos/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestSource(t *testing.T) {
	testutil.SetupGlobals(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	resultCh := make(chan runMirrorResult, 1)
	t.Cleanup(func() {
		close(resultCh)
	})

	go runDebianMirror(ctx, resultCh)

	mirrorResult := <-resultCh
	require.NoError(t, mirrorResult.err)

	s, err := source.NewSource(ctx, latestrecipe.SourceConfig{
		URL:      fmt.Sprintf("http://%s/debian", mirrorResult.addr.String()),
		SignedBy: filepath.Join(testutil.Root(), "testdata/archive-key-12.asc"),
	})
	require.NoError(t, err)

	components, err := s.Components(ctx, arch.MustParse("amd64"))
	require.NoError(t, err)

	require.Len(t, components, 2)
	require.Equal(t, "main", components[0].Name)
	require.Equal(t, "all", components[0].Arch.String())
	require.Equal(t, "main", components[1].Name)
	require.Equal(t, "amd64", components[1].Arch.String())

	componentPackages, lastUpdated, err := components[1].Packages(ctx)
	require.NoError(t, err)

	require.Len(t, componentPackages, 63408)

	require.NotEqual(t, time.Time{}, lastUpdated)
}

type runMirrorResult struct {
	err  error
	addr net.Addr
}

func runDebianMirror(ctx context.Context, result chan runMirrorResult) {
	mux := http.NewServeMux()

	rootDir := filepath.Join(testutil.Root(), "testdata")

	mux.HandleFunc("/debian/dists/stable/InRelease", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(rootDir, "InRelease"))
	})

	mux.HandleFunc("/debian/dists/stable/main/binary-amd64/Packages.gz", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(rootDir, "Packages.gz"))
	})

	srv := &http.Server{
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		result <- runMirrorResult{err: err}
		return
	}

	result <- runMirrorResult{addr: lis.Addr()}

	go func() {
		<-ctx.Done()

		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.Serve(lis); err != nil {
		return
	}
}
