/*
Copyright (c) 2023 - 2024 Purple Clay

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package nsv_test

import (
	"os"
	"testing"

	git "github.com/purpleclay/gitz"
	"github.com/purpleclay/gitz/gittest"
	"github.com/purpleclay/nsv/internal/nsv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextVersion(t *testing.T) {
	log := `(main, origin/main) docs: document new search improvements
fix(search): search is not being aggregated correctly
(tag: 0.1.0) feat(search): support aggregations for search analytics
ci: add parallel testing support to workflow`
	gittest.InitRepository(t, gittest.WithLog(log))
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{})
	require.NoError(t, err)

	assert.Equal(t, "0.1.1", next.Tag)
	assert.Equal(t, 1, next.Match.Index)
	assert.Equal(t, "fix(search): search is not being aggregated correctly", next.Log[next.Match.Index].Message)
	assert.Empty(t, next.LogDir)
}

func TestNextVersionFirstVersion(t *testing.T) {
	tests := []struct {
		name     string
		log      string
		expected string
	}{
		{
			name:     "Patch",
			log:      "fix: incorrectly displaying results in tui",
			expected: "0.0.1",
		},
		{
			name:     "Minor",
			log:      "feat: add new tui for displaying job summary",
			expected: "0.1.0",
		},
		{
			name:     "Major",
			log:      "feat!: somehow we have a breaking change for the first commit",
			expected: "0.1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gittest.InitRepository(t, gittest.WithLog(tt.log))
			gitc, _ := git.NewClient()

			next, err := nsv.NextVersion(gitc, nsv.Options{})

			require.NoError(t, err)
			require.Equal(t, tt.expected, next.Tag)
		})
	}
}

func TestNextVersionMajorZeroSemV(t *testing.T) {
	log := `(main, origin/main) refactor!: switch to v0.1 of the schema which has no backwards compatibility
(tag: 0.1.2) fix: issues with v0.0.8 of schema`
	gittest.InitRepository(t, gittest.WithLog(log))
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{})

	require.NoError(t, err)
	require.Equal(t, "0.2.0", next.Tag)
}

func TestNextVersionMajorZeroSemVForceMajor(t *testing.T) {
	log := `> (main, origin/main) feat: everything is now stable ready for v1
nsv:force~major
> (tag: 0.9.9) fix: stability issues around long running database connectivity`
	gittest.InitRepository(t, gittest.WithLog(log))
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{})

	require.NoError(t, err)
	require.Equal(t, "1.0.0", next.Tag)
}

func TestNextVersionIncludesSubDirectoryAsPrefix(t *testing.T) {
	log := `feat(trends)!: breaking change to capturing user trends`
	gittest.InitRepository(t, gittest.WithLog(log), gittest.WithFiles("src/search/main.go", "src/store/main.go"))
	gittest.StageFile(t, "src/search/main.go")
	gittest.Commit(t, "feat(search): add support to search across user trends for recommendations")
	gittest.StageFile(t, "src/store/main.go")
	gittest.Commit(t, "fix(store): fixed timestamp formatting issues")

	os.Chdir("src/search")
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{})

	require.NoError(t, err)
	assert.Equal(t, "search/0.1.0", next.Tag)
}

func TestNextVersionWithPath(t *testing.T) {
	log := `feat(ui)!: breaking change to search engine
feat(search): add ability to search across processed data`
	gittest.InitRepository(t, gittest.WithLog(log), gittest.WithFiles("src/processor/main.go"))
	gittest.StageFile(t, "src/processor/main.go")
	gittest.Commit(t, "feat(process): add support for processing bulk data")

	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{Path: "src/processor"})
	require.NoError(t, err)
	assert.Equal(t, "processor/0.1.0", next.Tag)
	assert.Equal(t, "src/processor", next.LogDir)
}

func TestNextVersionPreservesTagPrefix(t *testing.T) {
	tests := []struct {
		name     string
		log      string
		expected string
	}{
		{
			name: "VPrefix",
			log: `(main, origin/main) feat(ui): rebrand existing components with new theme
docs: update documentation with breaking change
(tag: v1.0.0) feat(ui)!: breaking redesign of search ui`,
			expected: "v1.1.0",
		},
		{
			name: "MonoRepoPrefix",
			log: `(HEAD -> main) fix(cache): incorrect sorting of metadata cache
docs: update documentation with latest cache improvement
ci: configure workflow to run benchmarks
(tag: cache/v0.2.0) feat(cache): `,
			expected: "cache/v0.2.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gittest.InitRepository(t, gittest.WithLog(tt.log))
			gitc, _ := git.NewClient()

			next, err := nsv.NextVersion(gitc, nsv.Options{})

			require.NoError(t, err)
			require.Equal(t, tt.expected, next.Tag)
		})
	}
}

func TestNextVersionFromPrerelease(t *testing.T) {
	log := `(main, origin/main) feat: api is now stable, flag has been removed
(tag: 0.2.0-beta.1) feat: experimental feature of api enabled with opt-in flag`
	gittest.InitRepository(t, gittest.WithLog(log))
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{})
	require.NoError(t, err)
	assert.Equal(t, "0.2.0", next.Tag)
}

func TestNextVersionPrerelease(t *testing.T) {
	log := `> (main, origin/main) feat: initial restructure of documents for improved elastic search
nsv:pre~alpha
> (tag: 0.2.0) feat: use the elastic scroll api to page results`
	gittest.InitRepository(t, gittest.WithLog(log))
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{})
	require.NoError(t, err)
	assert.Equal(t, "0.3.0-alpha.1", next.Tag)
}

func TestNextVersionIncrementsPrerelease(t *testing.T) {
	log := `> (main, origin/main) feat: add support for coping a file within the cache to a new location
nsv:pre
> (tag: 0.2.0-beta.1) feat: experimental file cache with configurable ttl
nsv:pre`
	gittest.InitRepository(t, gittest.WithLog(log))
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{})
	require.NoError(t, err)
	assert.Equal(t, "0.2.0-beta.2", next.Tag)
}

func TestNextVersionWithFormat(t *testing.T) {
	log := "(main) feat(broker): support asynchronous publishing to broker"
	format := "custom/v{{ .Version }}"

	gittest.InitRepository(t, gittest.WithLog(log))
	gitc, _ := git.NewClient()

	next, err := nsv.NextVersion(gitc, nsv.Options{VersionFormat: format})

	require.NoError(t, err)
	assert.Equal(t, "custom/v0.1.0", next.Tag)
}

func TestParseTag(t *testing.T) {
	tag, _ := nsv.ParseTag("store/v0.11.2-beta.1+20230207")
	assert.Equal(t, tag.Prefix, "store")
	assert.Equal(t, tag.SemVer, "0.11.2-beta.1+20230207")
	assert.Equal(t, tag.Version, "v0.11.2-beta.1+20230207")
	assert.Equal(t, tag.Raw, "store/v0.11.2-beta.1+20230207")
	assert.Equal(t, tag.Pre, "beta.1")
	assert.Equal(t, tag.Metadata, "20230207")
}
