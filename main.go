// git-workpool — deterministic workpool operations for isolated agent work.
//
// Runs as a git extension: `git workpool <command>`.
// The pool lives outside the repo, rooted at GIT_WORKPOOL_HOME,
// git config workpool.home, or $XDG_DATA_HOME/git-workpool.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// codenames — ordered list of adjective-animal pairs (Ubuntu-style).
// setup takes the first unused one; falls back to clone-N when exhausted.
var codenames = []string{
	"flirty-beaver", "jolly-otter", "sleepy-hawk", "brisk-fox", "mellow-panda",
	"zippy-lobster", "lucky-hedgehog", "nimble-falcon", "cosy-lynx", "plucky-badger",
	"sassy-wombat", "wobbly-flamingo", "peppy-narwhal", "goofy-puffin", "snappy-axolotl",
	"rosy-capybara", "chirpy-kookaburra", "feisty-manatee", "grumpy-pelican", "sunny-meerkat",
	"dizzy-dolphin", "fuzzy-armadillo", "jumpy-kangaroo", "dozy-dragonfly", "cranky-camel",
	"dreamy-dingo", "breezy-bison", "witty-walrus", "fancy-finch", "noble-newt",
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		usage()
		return
	}
	pool := poolRoot()

	switch cmd {
	case "setup", "status", "claim", "publish", "pull", "close":
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(1)
	}

	var err error
	switch cmd {
	case "setup":
		err = withRepo(pool, func(info repoInfo) error { return cmdSetup(pool, info) })
	case "status":
		err = withRepo(pool, func(info repoInfo) error { return cmdStatus(pool, info.project) })
	case "claim":
		force, branch, e := parseClaimArgs(os.Args[2:])
		if e != nil {
			err = e
			break
		}
		err = withRepo(pool, func(info repoInfo) error { return cmdClaim(pool, info.project, force, branch) })
	case "publish":
		branch := branchArg(os.Args[2:])
		err = withRepo(pool, func(info repoInfo) error { return cmdPublish(info, branch) })
	case "pull":
		branch := branchArg(os.Args[2:])
		err = withRepo(pool, func(info repoInfo) error { return cmdPull(info, branch) })
	case "close":
		err = withRepo(pool, func(info repoInfo) error { return cmdClose(pool, info) })
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// withRepo resolves the current repo once and hands it to fn.
func withRepo(pool string, fn func(repoInfo) error) error {
	info, err := currentRepo(pool)
	if err != nil {
		return err
	}
	return fn(info)
}

func usage() {
	fmt.Print(`git workpool — isolated workpool clones for agent work.

The pool lives outside the repo. Root resolution: $GIT_WORKPOOL_HOME, then
git config --global workpool.home, then $XDG_DATA_HOME/git-workpool.

Commands:
  setup                    create hub (first run), then one codenamed clone per call
  status                   show hub + clones: branch, busy/free, un-pushed/dirty
  claim [--force NAME] [BRANCH]
                           sync a free clone to BRANCH and print its folder.
                           --force NAME: rescue + reset + claim that clone (permission-gated)
  publish [BRANCH]         push current branch to the hub (main clone or workpool clone)
  pull [BRANCH]            main clone: fetch + merge the branch from the hub
  close                    workpool clone: discard local state, reset to main, free

A clone is free when clean and fully pushed to the hub. The hub is the only
link between your main clone and the workpool clones; the pool never touches
your remote.
`)
}

// poolRoot resolves the pool location in priority order.
func poolRoot() string {
	p := ""
	if v := os.Getenv("GIT_WORKPOOL_HOME"); v != "" {
		p = abs(v)
	} else if v, err := git("", "config", "--global", "--get", "workpool.home"); err == nil && v != "" {
		p = abs(v)
	} else if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		p = filepath.Join(abs(x), "git-workpool")
	} else if home, err := os.UserHomeDir(); err == nil {
		p = filepath.Join(home, ".local", "share", "git-workpool")
	} else {
		p = "git-workpool"
	}
	// canonicalize symlinks (e.g. macOS /var → /private/var) so the pool path
	// matches the real path git rev-parse reports from inside clones
	return resolveSymlinks(p)
}

// resolveSymlinks evals the deepest existing ancestor, then rejoins the rest.
func resolveSymlinks(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	parent := filepath.Dir(p)
	if parent == p {
		return p
	}
	return filepath.Join(resolveSymlinks(parent), filepath.Base(p))
}

func abs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

type repoInfo struct {
	root    string // repo root (toplevel)
	project string // project key = repo folder name
	isClone bool   // true when this repo is a workpool clone
	clone   string // clone codename (when isClone)
}

// currentRepo identifies the repo we're standing in and whether it's a clone.
func currentRepo(pool string) (repoInfo, error) {
	root, err := git("", "rev-parse", "--show-toplevel")
	if err != nil {
		return repoInfo{}, fmt.Errorf("not inside a git repo: %v", err)
	}
	// a workpool clone lives at <pool>/<project>/<codename>
	parent := filepath.Dir(root)
	if filepath.Dir(parent) == pool {
		return repoInfo{root: root, project: filepath.Base(parent), isClone: true, clone: filepath.Base(root)}, nil
	}
	return repoInfo{root: root, project: filepath.Base(root)}, nil
}

func projectDir(pool, project string) string { return filepath.Join(pool, project) }
func hubDir(pool, project string) string     { return filepath.Join(pool, project, "hub.git") }

// git runs git in dir, returning trimmed combined output.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func parseClaimArgs(args []string) (force, branch string, err error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--force" {
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--force requires a clone name (never auto-picked)")
			}
			i++
			force = args[i]
		} else {
			branch = args[i]
		}
	}
	return force, branch, nil
}

func branchArg(args []string) string {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0]
	}
	return ""
}

func listClones(pool, project string) ([]string, error) {
	entries, err := os.ReadDir(projectDir(pool, project))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "hub.git" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func nextCodename(pool, project string) string {
	used := map[string]bool{}
	if names, err := listClones(pool, project); err == nil {
		for _, n := range names {
			used[n] = true
		}
	}
	for _, c := range codenames {
		if !used[c] {
			return c
		}
	}
	for n := len(used) + 1; ; n++ {
		name := fmt.Sprintf("clone-%d", n)
		if !used[name] {
			return name
		}
	}
}

// defaultBranch returns the clone's default branch name (from origin/HEAD).
func defaultBranch(path string) string {
	if out, err := git(path, "rev-parse", "--abbrev-ref", "origin/HEAD"); err == nil && out != "" {
		return strings.TrimPrefix(out, "origin/")
	}
	return "main"
}

// cmdSetup creates the hub (first run) and one codenamed clone per call.
func cmdSetup(pool string, info repoInfo) error {
	if info.isClone {
		return fmt.Errorf("setup runs in the main clone, not inside a workpool clone")
	}
	pdir := projectDir(pool, info.project)
	hub := hubDir(pool, info.project)
	if _, err := os.Stat(hub); os.IsNotExist(err) {
		if err := os.MkdirAll(pdir, 0o755); err != nil {
			return err
		}
		if out, err := git("", "clone", "--bare", info.root, hub); err != nil {
			return fmt.Errorf("create hub: %v\n%s", err, out)
		}
		fmt.Printf("hub created: %s\n", hub)
	} else {
		fmt.Printf("hub exists: %s\n", hub)
	}
	// wire the main clone to the hub
	if _, err := git(info.root, "remote", "get-url", "hub"); err != nil {
		if out, err := git(info.root, "remote", "add", "hub", hub); err != nil {
			return fmt.Errorf("add hub remote: %v\n%s", err, out)
		}
		fmt.Printf("hub remote added\n")
	}
	name := nextCodename(pool, info.project)
	clonePath := filepath.Join(pdir, name)
	if _, err := os.Stat(clonePath); err == nil {
		return fmt.Errorf("clone %q already exists at %s", name, clonePath)
	}
	if out, err := git("", "clone", hub, clonePath); err != nil {
		return fmt.Errorf("create clone: %v\n%s", err, out)
	}
	installDeps(clonePath)
	fmt.Printf("clone created: %s (%s)\n", name, clonePath)
	return nil
}

func installDeps(path string) {
	pm := ""
	switch {
	case fileExists(path, "bun.lock"), fileExists(path, "bun.lockb"):
		pm = "bun"
	case fileExists(path, "pnpm-lock.yaml"):
		pm = "pnpm"
	case fileExists(path, "package-lock.json"):
		pm = "npm"
	case fileExists(path, "yarn.lock"):
		pm = "yarn"
	}
	if pm == "" {
		fmt.Println("deps: no lockfile found, skipping install")
		return
	}
	if _, err := exec.LookPath(pm); err != nil {
		fmt.Printf("deps: %s not installed, skipping\n", pm)
		return
	}
	cmd := exec.Command(pm, "install")
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("deps: %s install failed: %v\n%s", pm, err, out)
		return
	}
	fmt.Printf("deps: %s install done\n", pm)
}

func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

type cloneState struct {
	name   string
	path   string
	branch string
	dirty  int // uncommitted + untracked changes
	ahead  int // commits the hub doesn't have yet
	hasHub bool
}

func (s cloneState) busy() bool {
	if s.dirty > 0 {
		return true
	}
	if s.branch != defaultBranch(s.path) && !s.hasHub {
		return true
	}
	return s.ahead > 0
}

func cloneStates(pool, project string) []cloneState {
	names, err := listClones(pool, project)
	if err != nil {
		return nil
	}
	states := make([]cloneState, 0, len(names))
	for _, name := range names {
		path := filepath.Join(projectDir(pool, project), name)
		branch, _ := git(path, "rev-parse", "--abbrev-ref", "HEAD")
		s := cloneState{name: name, path: path, branch: branch}
		if out, err := git(path, "status", "--porcelain"); err == nil && out != "" {
			s.dirty = len(strings.Split(out, "\n"))
		}
		if out, err := git(path, "rev-parse", "--verify", "--quiet", "origin/"+branch); err == nil && out != "" {
			s.hasHub = true
			if n, err := git(path, "rev-list", "--count", "origin/"+branch+"..HEAD"); err == nil {
				fmt.Sscan(n, &s.ahead)
			}
		}
		states = append(states, s)
	}
	return states
}

// cmdStatus prints hub branches and per-clone state.
func cmdStatus(pool, project string) error {
	if _, err := os.Stat(projectDir(pool, project)); os.IsNotExist(err) {
		return fmt.Errorf("no pool for %q yet — run `git workpool setup`", project)
	}
	fmt.Printf("pool: %s\n", projectDir(pool, project))
	if _, err := os.Stat(hubDir(pool, project)); err == nil {
		if out, err := git(hubDir(pool, project), "for-each-ref", "--format=%(refname:short)", "refs/heads"); err == nil && out != "" {
			fmt.Printf("hub branches: %s\n", strings.Join(strings.Split(out, "\n"), ", "))
		}
	} else {
		fmt.Println("hub: missing — run `git workpool setup`")
	}
	for _, s := range cloneStates(pool, project) {
		mark := "free"
		if s.busy() {
			mark = "busy"
		}
		extra := ""
		if s.dirty > 0 {
			extra += fmt.Sprintf(" %ddirty", s.dirty)
		}
		if s.ahead > 0 {
			extra += fmt.Sprintf(" %dahead", s.ahead)
		}
		if s.branch != defaultBranch(s.path) && !s.hasHub {
			extra += " never-pushed"
		}
		fmt.Printf("  %-20s %-5s %s%s\n", s.name, mark, s.branch, extra)
	}
	return nil
}

// cmdClaim syncs a free clone to branch; --force NAME rescues + resets it first.
func cmdClaim(pool, project, force, branch string) error {
	states := cloneStates(pool, project)
	if len(states) == 0 {
		return fmt.Errorf("no clones in pool — run `git workpool setup`")
	}
	target, err := pickTarget(states, force)
	if err != nil {
		return err
	}
	if force != "" {
		if err := rescue(target); err != nil {
			return err
		}
	}
	// resolve branch: explicit, or re-engage the clone's existing branch
	if branch == "" {
		if target.branch != defaultBranch(target.path) && target.hasHub {
			branch = target.branch
		} else {
			return fmt.Errorf("no branch given and clone is on %q — pass a branch", target.branch)
		}
	}
	// sync: reset to hub's copy of the branch, or create from hub default
	if out, err := git(target.path, "fetch", "origin"); err != nil {
		return fmt.Errorf("fetch hub failed: %v\n%s", err, out)
	}
	if _, err := git(target.path, "rev-parse", "--verify", "--quiet", "origin/"+branch); err == nil {
		if out, err := git(target.path, "checkout", "-B", branch, "origin/"+branch); err != nil {
			return fmt.Errorf("checkout failed: %v\n%s", err, out)
		}
	} else {
		db := defaultBranch(target.path)
		if _, err := git(target.path, "rev-parse", "--verify", "--quiet", "origin/"+db); err != nil {
			return fmt.Errorf("hub has no %q branch — publish it from your main clone first: `git workpool publish %s`", db, db)
		}
		if out, err := git(target.path, "checkout", "-b", branch, "origin/"+db); err != nil {
			return fmt.Errorf("checkout failed: %v\n%s", err, out)
		}
	}
	fmt.Printf("claimed clone %s at %s (branch %s)\n", target.name, target.path, branch)
	return nil
}

func pickTarget(states []cloneState, force string) (*cloneState, error) {
	if force != "" {
		for i := range states {
			if states[i].name == force {
				return &states[i], nil
			}
		}
		return nil, fmt.Errorf("no clone named %q (available: %s)", force, cloneNames(states))
	}
	for i := range states {
		if !states[i].busy() {
			return &states[i], nil
		}
	}
	return nil, fmt.Errorf("all clones busy — run `git workpool status`, then close one or claim --force <name>")
}

func cloneNames(states []cloneState) string {
	names := make([]string, len(states))
	for i, s := range states {
		names[i] = s.name
	}
	return strings.Join(names, ", ")
}

// rescue preserves un-pushed commits (push to hub) and dirty files (stash).
func rescue(s *cloneState) error {
	fmt.Printf("force-claiming %s (busy on %s)\n", s.name, s.branch)
	// count commits the hub doesn't have yet
	rescued := 0
	if out, err := git(s.path, "rev-list", "--count", "HEAD", "--not", "--remotes"); err == nil {
		fmt.Sscan(out, &rescued)
	}
	if rescued > 0 {
		if out, err := git(s.path, "push", "origin", s.branch); err != nil {
			return fmt.Errorf("rescue push failed — refusing to force: %v\n%s", err, out)
		}
		fmt.Printf("  rescued %d commit(s) to hub on %s\n", rescued, s.branch)
	}
	if s.dirty > 0 {
		if out, err := git(s.path, "stash", "push", "-m", "workpool force-claim rescue"); err != nil {
			return fmt.Errorf("rescue stash failed — refusing to force: %v\n%s", err, out)
		}
		fmt.Printf("  stashed %d change(s) as stash@{0} in %s\n", s.dirty, s.name)
	}
	return nil
}

// cmdPublish pushes the current branch to the hub. Never commits.
// In the main clone the branch exists only as a hub ref, so push HEAD:<branch>.
func cmdPublish(info repoInfo, branch string) error {
	if branch == "" {
		branch, _ = git(info.root, "rev-parse", "--abbrev-ref", "HEAD")
	}
	if info.isClone {
		if _, err := git(info.root, "remote", "get-url", "origin"); err != nil {
			return fmt.Errorf("no origin remote — run `git workpool setup` first")
		}
		out, err := git(info.root, "push", "origin", branch)
		if out != "" {
			fmt.Println(out)
		}
		if err != nil {
			return fmt.Errorf("publish failed: %v", err)
		}
		return nil
	}
	if _, err := git(info.root, "remote", "get-url", "hub"); err != nil {
		return fmt.Errorf("no hub remote — run `git workpool setup` first")
	}
	out, err := git(info.root, "push", "hub", "HEAD:"+branch)
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}
	return nil
}

// cmdPull fetches and merges the branch from the hub into the main clone.
func cmdPull(info repoInfo, branch string) error {
	if info.isClone {
		return fmt.Errorf("pull runs in the main clone, not inside a workpool clone")
	}
	if _, err := git(info.root, "remote", "get-url", "hub"); err != nil {
		return fmt.Errorf("no hub remote — run `git workpool setup` first")
	}
	if branch == "" {
		branch, _ = git(info.root, "rev-parse", "--abbrev-ref", "HEAD")
	}
	if out, err := git(info.root, "fetch", "hub"); err != nil {
		return fmt.Errorf("fetch hub failed: %v\n%s", err, out)
	}
	out, err := git(info.root, "merge", "hub/"+branch)
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		return fmt.Errorf("merge failed — resolve conflicts, then commit: %v", err)
	}
	return nil
}

// cmdClose discards local clone state and resets to the hub default branch.
func cmdClose(pool string, info repoInfo) error {
	if !info.isClone {
		return fmt.Errorf("close runs inside a workpool clone")
	}
	states := cloneStates(pool, info.project)
	var s *cloneState
	for i := range states {
		if states[i].name == info.clone {
			s = &states[i]
			break
		}
	}
	if s == nil {
		return fmt.Errorf("%s is not a workpool clone", info.clone)
	}
	fmt.Printf("closing %s (on %s):", s.name, s.branch)
	switch {
	case s.dirty > 0 && s.ahead > 0:
		fmt.Printf(" %d change(s) and %d commit(s) will be discarded", s.dirty, s.ahead)
	case s.dirty > 0:
		fmt.Printf(" %d change(s) will be discarded", s.dirty)
	case s.ahead > 0:
		fmt.Printf(" %d commit(s) will be discarded", s.ahead)
	default:
		fmt.Print(" clean, nothing to discard")
	}
	fmt.Println()
	db := defaultBranch(s.path)
	if out, err := git(s.path, "fetch", "origin"); err != nil {
		return fmt.Errorf("fetch failed: %v\n%s", err, out)
	}
	if out, err := git(s.path, "checkout", "-B", db, "origin/"+db); err != nil {
		return fmt.Errorf("reset failed: %v\n%s", err, out)
	}
	if out, err := git(s.path, "clean", "-fd"); err != nil {
		return fmt.Errorf("clean failed: %v\n%s", err, out)
	}
	fmt.Printf("clone %s free\n", s.name)
	return nil
}
