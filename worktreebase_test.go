//
// Date: 2026-10-16
// Author: cgill27 (cgill27@gmail.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeGit is a scripted gitRunner: each response is keyed by the git args joined
// with spaces, and every call is recorded so tests can assert what ran. Like
// exec.CommandContext it refuses to run under a context that is already done,
// and a key in hang blocks until its context expires (a stalled network call).
type fakeGit struct {
	out   map[string]string
	fail  map[string]bool
	hang  map[string]bool
	calls []string
}

func (f *fakeGit) run(ctx context.Context, _ string, args ...string) (string, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if f.hang[key] {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if f.fail[key] {
		return "", errors.New("exit status 1")
	}
	return f.out[key], nil
}

func (f *fakeGit) ran(key string) bool {
	for _, c := range f.calls {
		if c == key {
			return true
		}
	}
	return false
}

func (f *fakeGit) resolver() worktreeBaseResolver {
	return worktreeBaseResolver{
		git:          f.run,
		lookPath:     func(string) (string, error) { return "/usr/bin/git", nil },
		timeout:      time.Second,
		localTimeout: time.Second,
	}
}

// TestResolveWorktreeBaseEmpty confirms an unset base runs no git at all and is
// omitted, keeping herdr's HEAD default.
func TestResolveWorktreeBaseEmpty(t *testing.T) {
	g := &fakeGit{}
	ref, warn := g.resolver().resolve("/repo", "  ", true)
	if ref != "" || warn != "" {
		t.Fatalf("resolve = (%q, %q), want empty", ref, warn)
	}
	if len(g.calls) != 0 {
		t.Fatalf("ran git %v, want nothing", g.calls)
	}
}

// TestResolveWorktreeBaseOriginDefault confirms origin/default asks the remote
// for its default branch, preferring it over a stale recorded origin/HEAD, and
// fetches that branch.
func TestResolveWorktreeBaseOriginDefault(t *testing.T) {
	g := &fakeGit{out: map[string]string{
		"symbolic-ref --short refs/remotes/origin/HEAD": "origin/master",
		"ls-remote --symref origin HEAD":                "ref: refs/heads/main\tHEAD\nabc123\tHEAD",
		"remote":                                        "origin",
	}}
	ref, warn := g.resolver().resolve("/repo", worktreeBaseDefault, true)
	if ref != "origin/main" || warn != "" {
		t.Fatalf("resolve = (%q, %q), want (origin/main, \"\")", ref, warn)
	}
	if !g.ran("fetch --quiet --no-tags origin +refs/heads/main:refs/remotes/origin/main") {
		t.Fatalf("expected a fetch of origin main, calls=%v", g.calls)
	}
}

// TestResolveWorktreeBaseOriginDefaultNoFetch confirms that with fetch off a
// recorded origin/HEAD is used as-is, without touching the network.
func TestResolveWorktreeBaseOriginDefaultNoFetch(t *testing.T) {
	g := &fakeGit{out: map[string]string{
		"symbolic-ref --short refs/remotes/origin/HEAD": "origin/main",
		"remote": "origin",
	}}
	ref, warn := g.resolver().resolve("/repo", worktreeBaseDefault, false)
	if ref != "origin/main" || warn != "" {
		t.Fatalf("resolve = (%q, %q), want (origin/main, \"\")", ref, warn)
	}
	for _, call := range g.calls {
		if strings.HasPrefix(call, "fetch ") || strings.HasPrefix(call, "ls-remote") {
			t.Fatalf("network call %q with fetch off", call)
		}
	}
}

// TestResolveWorktreeBaseOriginDefaultOffline confirms an unreachable remote
// keeps the recorded origin/HEAD and still uses the last-fetched branch.
func TestResolveWorktreeBaseOriginDefaultOffline(t *testing.T) {
	g := &fakeGit{
		out: map[string]string{
			"symbolic-ref --short refs/remotes/origin/HEAD": "origin/main",
			"remote": "origin",
		},
		fail: map[string]bool{
			"ls-remote --symref origin HEAD":                                           true,
			"fetch --quiet --no-tags origin +refs/heads/main:refs/remotes/origin/main": true,
		},
	}
	ref, warn := g.resolver().resolve("/repo", worktreeBaseDefault, true)
	if ref != "origin/main" || !strings.Contains(warn, "last-fetched") {
		t.Fatalf("resolve = (%q, %q), want origin/main with a fetch warning", ref, warn)
	}
}

// TestResolveWorktreeBaseMissingOriginHead confirms a repo with no recorded
// origin/HEAD still finds the default branch by asking the remote, even with
// fetch off.
func TestResolveWorktreeBaseMissingOriginHead(t *testing.T) {
	g := &fakeGit{
		out: map[string]string{
			"ls-remote --symref origin HEAD": "ref: refs/heads/trunk\tHEAD\nabc123\tHEAD",
			"remote":                         "origin",
		},
		fail: map[string]bool{"symbolic-ref --short refs/remotes/origin/HEAD": true},
	}
	ref, warn := g.resolver().resolve("/repo", worktreeBaseDefault, false)
	if ref != "origin/trunk" || warn != "" {
		t.Fatalf("resolve = (%q, %q), want (origin/trunk, \"\")", ref, warn)
	}
}

// TestSymrefHeadBranch confirms only a refs/heads/ target is taken as origin's
// default branch.
func TestSymrefHeadBranch(t *testing.T) {
	cases := map[string]string{
		"ref: refs/heads/main\tHEAD\nabc123\tHEAD": "main",
		"ref: refs/heads/feat/x\tHEAD":             "feat/x",
		"ref: refs/tags/v1\tHEAD\nabc123\tHEAD":    "",
		"abc123\tHEAD":                             "",
		"":                                         "",
	}
	for out, want := range cases {
		if got := symrefHeadBranch(out); got != want {
			t.Errorf("symrefHeadBranch(%q) = %q, want %q", out, got, want)
		}
	}
}

// TestResolveWorktreeBaseNoOrigin confirms a repo without origin falls back to
// HEAD with a warning instead of failing.
func TestResolveWorktreeBaseNoOrigin(t *testing.T) {
	g := &fakeGit{fail: map[string]bool{
		"symbolic-ref --short refs/remotes/origin/HEAD": true,
		"ls-remote --symref origin HEAD":                true,
	}}
	ref, warn := g.resolver().resolve("/repo", worktreeBaseDefault, true)
	if ref != "" || !strings.Contains(warn, "from HEAD") {
		t.Fatalf("resolve = (%q, %q), want HEAD fallback warning", ref, warn)
	}
}

// TestResolveWorktreeBaseFetchFails confirms an offline fetch still uses the
// last-fetched remote ref, with a warning.
func TestResolveWorktreeBaseFetchFails(t *testing.T) {
	g := &fakeGit{
		out:  map[string]string{"remote": "origin"},
		fail: map[string]bool{"fetch --quiet --no-tags origin +refs/heads/main:refs/remotes/origin/main": true},
	}
	ref, warn := g.resolver().resolve("/repo", "origin/main", true)
	if ref != "origin/main" || !strings.Contains(warn, "last-fetched") {
		t.Fatalf("resolve = (%q, %q), want origin/main with a fetch warning", ref, warn)
	}
}

// TestResolveWorktreeBaseFetchTimeout confirms a fetch that uses up the whole
// network timeout still leaves the local lookups working, so the last-fetched
// ref is used rather than falling back to HEAD.
func TestResolveWorktreeBaseFetchTimeout(t *testing.T) {
	g := &fakeGit{
		out:  map[string]string{"remote": "origin"},
		hang: map[string]bool{"fetch --quiet --no-tags origin +refs/heads/main:refs/remotes/origin/main": true},
	}
	r := g.resolver()
	r.timeout = 20 * time.Millisecond
	ref, warn := r.resolve("/repo", "origin/main", true)
	if ref != "origin/main" || !strings.Contains(warn, "last-fetched") {
		t.Fatalf("resolve = (%q, %q), want origin/main with a fetch warning", ref, warn)
	}
	if !g.ran("rev-parse --verify --quiet origin/main^{commit}") {
		t.Fatalf("expected the local ref check to run after the timeout, calls=%v", g.calls)
	}
}

// TestResolveWorktreeBaseRejectsMalformedRef confirms a base that isn't a valid
// ref never reaches git fetch, where it could act as a refspec writing to some
// other ref, and falls back to HEAD with a warning.
func TestResolveWorktreeBaseRejectsMalformedRef(t *testing.T) {
	for _, base := range []string{"origin/main:main", "origin/+main:refs/heads/x", "origin/a..b", "--upload-pack=x"} {
		t.Run(base, func(t *testing.T) {
			branch := strings.TrimPrefix(base, "origin/")
			g := &fakeGit{
				out:  map[string]string{"remote": "origin"},
				fail: map[string]bool{"check-ref-format --branch " + branch: true},
			}
			ref, warn := g.resolver().resolve("/repo", base, true)
			if ref != "" || !strings.Contains(warn, "not a valid ref") {
				t.Fatalf("resolve = (%q, %q), want HEAD fallback with an invalid-ref warning", ref, warn)
			}
			for _, call := range g.calls {
				if strings.HasPrefix(call, "fetch ") {
					t.Fatalf("fetch ran for malformed base: %q", call)
				}
			}
		})
	}
}

// TestResolveWorktreeBaseUnresolvableRef confirms a ref that doesn't exist falls
// back to HEAD with a warning.
func TestResolveWorktreeBaseUnresolvableRef(t *testing.T) {
	g := &fakeGit{fail: map[string]bool{"rev-parse --verify --quiet nope^{commit}": true}}
	ref, warn := g.resolver().resolve("/repo", "nope", true)
	if ref != "" || !strings.Contains(warn, "from HEAD") {
		t.Fatalf("resolve = (%q, %q), want HEAD fallback warning", ref, warn)
	}
}

// TestResolveWorktreeBaseExplicitRef confirms explicit refs pass through: a
// local branch is never fetched, a remote one is fetched unless fetch is off,
// and the longest matching remote name wins.
func TestResolveWorktreeBaseExplicitRef(t *testing.T) {
	cases := []struct {
		name      string
		base      string
		fetch     bool
		wantFetch string
	}{
		{"local branch", "main", true, ""},
		{"remote branch", "origin/develop", true, "fetch --quiet --no-tags origin +refs/heads/develop:refs/remotes/origin/develop"},
		{"fetch off", "origin/develop", false, ""},
		{"nested remote", "up/stream/main", true, "fetch --quiet --no-tags up/stream +refs/heads/main:refs/remotes/up/stream/main"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := &fakeGit{out: map[string]string{"remote": "origin\nup\nup/stream"}}
			ref, warn := g.resolver().resolve("/repo", c.base, c.fetch)
			if ref != c.base || warn != "" {
				t.Fatalf("resolve = (%q, %q), want (%q, \"\")", ref, warn, c.base)
			}
			fetched := ""
			for _, call := range g.calls {
				if strings.HasPrefix(call, "fetch ") {
					fetched = call
				}
			}
			if fetched != c.wantFetch {
				t.Fatalf("fetch call = %q, want %q (calls=%v)", fetched, c.wantFetch, g.calls)
			}
		})
	}
}

// TestResolveWorktreeBaseGitNotFound confirms a missing git skips resolution
// entirely, falling back to HEAD with a warning that says why.
func TestResolveWorktreeBaseGitNotFound(t *testing.T) {
	g := &fakeGit{}
	r := g.resolver()
	r.lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	ref, warn := r.resolve("/repo", worktreeBaseDefault, true)
	if ref != "" || !strings.Contains(warn, "git not found") {
		t.Fatalf("resolve = (%q, %q), want HEAD fallback with a git-not-found warning", ref, warn)
	}
	if len(g.calls) != 0 {
		t.Fatalf("ran git %v without a git", g.calls)
	}
}

// TestResolveWorktreeBaseLocalTimeout confirms a local git call that hangs is
// cut off by its own timeout and falls back to HEAD with a warning.
func TestResolveWorktreeBaseLocalTimeout(t *testing.T) {
	g := &fakeGit{hang: map[string]bool{"rev-parse --verify --quiet main^{commit}": true}}
	r := g.resolver()
	r.localTimeout = 20 * time.Millisecond
	ref, warn := r.resolve("/repo", "main", true)
	if ref != "" || !strings.Contains(warn, "deadline exceeded") {
		t.Fatalf("resolve = (%q, %q), want HEAD fallback with a timeout warning", ref, warn)
	}
}

// fakeGitOnPath puts a `git` shell script with the given body first on PATH,
// shadowing the real one for the rest of the test; the rest of PATH stays so the
// script can still run sleep and friends.
func fakeGitOnPath(t *testing.T, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake git needs a POSIX shell")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestExecGitNotFound confirms execGit reports a missing git as errGitNotFound.
func TestExecGitNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := execGit(context.Background(), t.TempDir(), "status"); !errors.Is(err, errGitNotFound) {
		t.Fatalf("execGit err = %v, want errGitNotFound", err)
	}
}

// TestExecGitTimeout confirms a hung git is killed at the deadline and reported
// as a timeout. The fake git leaves a child (sleep) holding stdout, like ssh
// under git fetch, so this also checks execGit doesn't wait on that pipe.
func TestExecGitTimeout(t *testing.T) {
	fakeGitOnPath(t, "sleep 30")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := execGit(ctx, t.TempDir(), "fetch")
	if err == nil || !strings.Contains(err.Error(), "git fetch timed out") {
		t.Fatalf("execGit err = %v, want a timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond+gitWaitDelay+time.Second {
		t.Fatalf("execGit took %s to give up", elapsed)
	}
}

// TestResolveWorktreeBaseGitMissingFromPath runs the production resolver with
// no git on PATH.
func TestResolveWorktreeBaseGitMissingFromPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	ref, warn := resolveWorktreeBase(t.TempDir(), "origin/main", true)
	if ref != "" || !strings.Contains(warn, "git not found") {
		t.Fatalf("resolve = (%q, %q), want HEAD fallback with a git-not-found warning", ref, warn)
	}
}

// TestResolveWorktreeBaseHungGit runs the resolver over a real (fake-script) git
// that never answers, and confirms it gives up promptly and falls back to HEAD.
func TestResolveWorktreeBaseHungGit(t *testing.T) {
	fakeGitOnPath(t, "sleep 30")
	r := worktreeBaseResolver{
		git:          execGit,
		lookPath:     exec.LookPath,
		timeout:      100 * time.Millisecond,
		localTimeout: 100 * time.Millisecond,
	}
	start := time.Now()
	ref, warn := r.resolve(t.TempDir(), worktreeBaseDefault, true)
	if ref != "" || !strings.Contains(warn, "timed out") {
		t.Fatalf("resolve = (%q, %q), want HEAD fallback with a timeout warning", ref, warn)
	}
	// symbolic-ref and ls-remote each hit their deadline; allow for WaitDelay.
	if elapsed := time.Since(start); elapsed > 2*(100*time.Millisecond+gitWaitDelay)+time.Second {
		t.Fatalf("resolve took %s to give up", elapsed)
	}
}

// gitFixture is a bare "origin", a "seed" clone that pushes to it, and a "clone"
// standing in for the project's primary checkout.
type gitFixture struct {
	t                   *testing.T
	origin, seed, clone string
}

// newGitFixture builds the fixture with one commit on main, cloned once.
func newGitFixture(t *testing.T) *gitFixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	f := &gitFixture{t: t,
		origin: filepath.Join(root, "origin.git"),
		seed:   filepath.Join(root, "seed"),
		clone:  filepath.Join(root, "clone"),
	}
	f.git(root, "init", "--quiet", "--bare", "--initial-branch=main", f.origin)
	f.git(root, "init", "--quiet", "--initial-branch=main", f.seed)
	f.git(f.seed, "commit", "--quiet", "--allow-empty", "-m", "one")
	f.git(f.seed, "remote", "add", "origin", f.origin)
	f.git(f.seed, "push", "--quiet", "origin", "main")
	f.git(root, "clone", "--quiet", f.origin, f.clone)
	return f
}

// git runs git in dir with a fixed identity and no user config, failing the
// test on error, and returns trimmed output.
func (f *gitFixture) git(dir string, args ...string) string {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestResolveWorktreeBaseRealRepo runs the real git against a clone whose
// origin has moved on since it last fetched, and confirms origin/default lands
// on origin's new tip rather than the clone's stale view of it.
func TestResolveWorktreeBaseRealRepo(t *testing.T) {
	f := newGitFixture(t)

	// origin moves on, with a tag on the new commit; the clone hasn't fetched it.
	f.git(f.seed, "commit", "--quiet", "--allow-empty", "-m", "two")
	f.git(f.seed, "tag", "v2")
	f.git(f.seed, "push", "--quiet", "origin", "main", "v2")
	want := f.git(f.seed, "rev-parse", "HEAD")
	if stale := f.git(f.clone, "rev-parse", "origin/main"); stale == want {
		t.Fatal("test setup: clone already has origin's tip")
	}

	ref, warn := resolveWorktreeBase(f.clone, worktreeBaseDefault, true)
	if ref != "origin/main" || warn != "" {
		t.Fatalf("resolve = (%q, %q), want (origin/main, \"\")", ref, warn)
	}
	if got := f.git(f.clone, "rev-parse", "origin/main"); got != want {
		t.Fatalf("origin/main = %s after resolve, want origin's tip %s", got, want)
	}
	// The fetch only updates origin/main: tags on the fetched commit aren't
	// followed into the clone.
	if tags := f.git(f.clone, "tag", "--list"); tags != "" {
		t.Fatalf("fetch created tags %q, want none", tags)
	}
}

// TestResolveWorktreeBaseRealRepoDefaultBranchChanged confirms origin/default
// follows the remote when its default branch changes after the clone was made,
// instead of trusting the clone's recorded origin/HEAD.
func TestResolveWorktreeBaseRealRepoDefaultBranchChanged(t *testing.T) {
	f := newGitFixture(t)

	f.git(f.seed, "switch", "--quiet", "-c", "trunk")
	f.git(f.seed, "commit", "--quiet", "--allow-empty", "-m", "on trunk")
	f.git(f.seed, "push", "--quiet", "origin", "trunk")
	f.git(f.origin, "symbolic-ref", "HEAD", "refs/heads/trunk")
	want := f.git(f.seed, "rev-parse", "HEAD")

	ref, warn := resolveWorktreeBase(f.clone, worktreeBaseDefault, true)
	if ref != "origin/trunk" || warn != "" {
		t.Fatalf("resolve = (%q, %q), want (origin/trunk, \"\")", ref, warn)
	}
	if got := f.git(f.clone, "rev-parse", "origin/trunk"); got != want {
		t.Fatalf("origin/trunk = %s, want %s", got, want)
	}
}

// TestResolveWorktreeBaseRealRepoMalformedRef confirms a base that would read as
// a refspec ("origin/main:side") is rejected before fetch, so no local branch is
// created or moved.
func TestResolveWorktreeBaseRealRepoMalformedRef(t *testing.T) {
	f := newGitFixture(t)

	ref, warn := resolveWorktreeBase(f.clone, "origin/main:side", true)
	if ref != "" || !strings.Contains(warn, "not a valid ref") {
		t.Fatalf("resolve = (%q, %q), want HEAD fallback with an invalid-ref warning", ref, warn)
	}
	if out := f.git(f.clone, "branch", "--list", "side"); out != "" {
		t.Fatalf("fetch created local branch %q", out)
	}
}
