//
// Date: 2026-10-16
// Author: cgill27 (cgill27@gmail.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// worktreeBaseDefault is the [worktree] base value that means "the remote's
// default branch": origin/HEAD resolved to e.g. origin/main.
const worktreeBaseDefault = "origin/default"

// worktreeFetchTimeout bounds the network git calls (ls-remote, fetch) run
// before creating a worktree, together, so a hung network can't hold the picker
// pane open indefinitely.
const worktreeFetchTimeout = 15 * time.Second

// gitLocalTimeout bounds each local git call (rev-parse, symbolic-ref, …). They
// normally take milliseconds; the bound only matters for a wedged repo or
// filesystem.
const gitLocalTimeout = 5 * time.Second

// gitWaitDelay is how long execGit waits for git's output to close after the
// context kills it. A fetch over ssh leaves an ssh child holding the pipe, and
// without this the wait would outlive the timeout.
const gitWaitDelay = time.Second

// errGitNotFound is returned by execGit when there is no git on PATH.
var errGitNotFound = errors.New("git not found in PATH")

// gitRunner runs `git -C dir args...` and returns its trimmed stdout. It is a
// variable on the resolver so tests can stand in a fake git.
type gitRunner func(ctx context.Context, dir string, args ...string) (string, error)

// execGit is the real gitRunner. It never prompts (GIT_TERMINAL_PROMPT=0, stdin
// from /dev/null, and on Unix no controlling terminal — see isolateGit), reports
// a missing git as errGitNotFound, and when ctx expires kills git and its
// children and returns an error naming the timeout instead of a bare
// "signal: killed".
func execGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.WaitDelay = gitWaitDelay
	isolateGit(cmd)
	out, err := cmd.Output()
	switch {
	case err == nil:
	case errors.Is(err, exec.ErrNotFound):
		err = errGitNotFound
	case ctx.Err() != nil:
		err = fmt.Errorf("git %s timed out: %w", args[0], ctx.Err())
	}
	return strings.TrimSpace(string(out)), err
}

// worktreeBaseResolver turns the configured [worktree] base into the ref handed
// to herdr's worktree.create.
type worktreeBaseResolver struct {
	git          gitRunner
	lookPath     func(file string) (string, error)
	timeout      time.Duration // budget shared by the network calls
	localTimeout time.Duration // bound on each local call
}

// resolveWorktreeBase is the production entry point; see
// worktreeBaseResolver.resolve.
func resolveWorktreeBase(dir, base string, fetch bool) (ref, warn string) {
	r := worktreeBaseResolver{
		git:          execGit,
		lookPath:     exec.LookPath,
		timeout:      worktreeFetchTimeout,
		localTimeout: gitLocalTimeout,
	}
	return r.resolve(dir, base, fetch)
}

// localGit runs a local git call under its own localTimeout, so one slow call
// can't eat into the budget of the next.
func (r worktreeBaseResolver) localGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.localTimeout)
	defer cancel()
	return r.git(ctx, dir, args...)
}

// resolve returns the ref new worktree branches should start from, and a warning
// for the user when something went wrong along the way.
//
//   - base "" returns "" so the base is omitted and herdr uses HEAD of dir — the
//     original behaviour.
//   - base "origin/default" resolves origin's default branch, e.g. origin/main.
//     When fetching, it asks the remote so a renamed default branch is picked
//     up; offline, or with fetch off, the recorded origin/HEAD is used.
//   - anything else is used as an explicit ref, e.g. "origin/develop" or "main".
//
// When fetch is true and the ref is a remote-tracking branch (<remote>/<branch>
// for a configured remote), that branch is fetched first, with an explicit
// refspec and --no-tags so the fetch can only update refs/remotes/<remote>/<branch>. A failed
// or timed-out fetch keeps the last-fetched remote ref, with a warning. If the
// ref is malformed or can't be resolved at all, resolve returns "" with a
// warning, so the worktree is still created from HEAD: creating a worktree is
// never blocked by the network or the remote.
//
// git missing from PATH skips all of this with a warning. The network calls
// (ls-remote, fetch) share one timeout; each local call has its own, so a fetch
// that uses up the network budget can't also fail the local lookups after it.
func (r worktreeBaseResolver) resolve(dir, base string, fetch bool) (ref, warn string) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", ""
	}

	if _, err := r.lookPath("git"); err != nil {
		return "", fmt.Sprintf("%v; creating the worktree from HEAD", errGitNotFound)
	}

	network, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	ref = base
	if base == worktreeBaseDefault {
		var err error
		ref, err = r.originHead(network, dir, fetch)
		if err != nil {
			return "", fmt.Sprintf("could not resolve origin's default branch (%v); creating the worktree from HEAD", err)
		}
	}
	if strings.HasPrefix(ref, "-") {
		return "", fmt.Sprintf("base %q is not a valid ref; creating the worktree from HEAD", ref)
	}

	var warns []string
	if remote, branch, ok := r.splitRemoteRef(dir, ref); ok {
		// Reject anything git wouldn't accept as a branch name (":", "..", a
		// leading "+", …) before it can reach fetch as part of a refspec.
		if _, err := r.localGit(dir, "check-ref-format", "--branch", branch); err != nil {
			return "", fmt.Sprintf("base %q is not a valid ref; creating the worktree from HEAD", ref)
		}
		if fetch {
			refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branch, remote, branch)
			if _, err := r.git(network, dir, "fetch", "--quiet", "--no-tags", remote, refspec); err != nil {
				warns = append(warns, fmt.Sprintf("git fetch %s %s failed (%v); using the last-fetched %s", remote, branch, err, ref))
			}
		}
	}

	if _, err := r.localGit(dir, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); err != nil {
		warns = append(warns, fmt.Sprintf("base %q could not be resolved to a commit (%v); creating the worktree from HEAD", ref, err))
		return "", strings.Join(warns, "; ")
	}
	return ref, strings.Join(warns, "; ")
}

// originHead returns origin's default branch as a remote-tracking ref, e.g.
// "origin/main". A clone records it as refs/remotes/origin/HEAD, but that goes
// stale if the remote's default branch is renamed, and a repo that added origin
// by hand may not have it at all. So when refresh is set, or nothing is
// recorded, it asks the remote (`git ls-remote --symref origin HEAD`, bounded by
// network), falling back to the recorded value if the remote can't be reached.
// ls-remote is used rather than `git remote set-head --auto`, which refuses a
// default branch that hasn't been fetched yet — exactly the renamed case.
func (r worktreeBaseResolver) originHead(network context.Context, dir string, refresh bool) (string, error) {
	recorded, err := r.localGit(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		recorded = ""
	}
	if recorded != "" && !refresh {
		return recorded, nil
	}
	out, err := r.git(network, dir, "ls-remote", "--symref", "origin", "HEAD")
	if err == nil {
		if branch := symrefHeadBranch(out); branch != "" {
			return "origin/" + branch, nil
		}
	}
	if recorded != "" {
		return recorded, nil
	}
	if err != nil {
		return "", fmt.Errorf("git ls-remote --symref origin HEAD: %w", err)
	}
	return "", fmt.Errorf("origin does not advertise a default branch")
}

// symrefHeadBranch extracts the branch from `git ls-remote --symref <remote>
// HEAD` output, whose first line reads "ref: refs/heads/main\tHEAD". A HEAD
// pointing outside refs/heads/ yields "".
func symrefHeadBranch(out string) string {
	for _, line := range strings.Split(out, "\n") {
		target, name, ok := strings.Cut(strings.TrimPrefix(line, "ref: "), "\t")
		if !ok || name != "HEAD" || !strings.HasPrefix(line, "ref: ") {
			continue
		}
		// Only a branch can be a default branch; anything else is ignored.
		if branch, isBranch := strings.CutPrefix(target, "refs/heads/"); isBranch {
			return branch
		}
		return ""
	}
	return ""
}

// splitRemoteRef reports whether ref names a branch on a configured remote
// ("origin/main" → "origin", "main"). The longest matching remote name wins, so
// a remote called "up/stream" isn't mistaken for "up".
func (r worktreeBaseResolver) splitRemoteRef(dir, ref string) (remote, branch string, ok bool) {
	out, err := r.localGit(dir, "remote")
	if err != nil {
		return "", "", false
	}
	for _, name := range strings.Fields(out) {
		if strings.HasPrefix(ref, name+"/") && len(name) > len(remote) {
			remote, branch = name, strings.TrimPrefix(ref, name+"/")
		}
	}
	return remote, branch, remote != "" && branch != ""
}
