// Package command implements the git workpool subcommands.
package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/avrebarra/git-workpool/internal/clone"
	"github.com/avrebarra/git-workpool/internal/gitx"
	"github.com/avrebarra/git-workpool/internal/pool"
)

// Setup creates the hub (first run) and one codenamed clone per call.
func Setup(poolRoot string, info pool.Info) error {
	if info.IsClone {
		return fmt.Errorf("setup runs in the main clone, not inside a workpool clone")
	}
	pdir := pool.ProjectDir(poolRoot, info.Project)
	hub := pool.HubDir(poolRoot, info.Project)
	if _, err := os.Stat(hub); os.IsNotExist(err) {
		if err := os.MkdirAll(pdir, 0o755); err != nil {
			return err
		}
		if out, err := gitx.Run("", "clone", "--bare", info.Root, hub); err != nil {
			return fmt.Errorf("create hub: %v\n%s", err, out)
		}
		fmt.Printf("hub created: %s\n", hub)
	} else {
		fmt.Printf("hub exists: %s\n", hub)
	}
	// wire the main clone to the hub
	if _, err := gitx.Run(info.Root, "remote", "get-url", "hub"); err != nil {
		if out, err := gitx.Run(info.Root, "remote", "add", "hub", hub); err != nil {
			return fmt.Errorf("add hub remote: %v\n%s", err, out)
		}
		fmt.Printf("hub remote added\n")
	}
	name := clone.NextCodename(poolRoot, info.Project)
	clonePath := filepath.Join(pdir, name)
	if _, err := os.Stat(clonePath); err == nil {
		return fmt.Errorf("clone %q already exists at %s", name, clonePath)
	}
	if out, err := gitx.Run("", "clone", hub, clonePath); err != nil {
		return fmt.Errorf("create clone: %v\n%s", err, out)
	}
	installDeps(clonePath)
	fmt.Printf("clone created: %s (%s)\n", name, clonePath)
	return nil
}

// Status prints hub branches and per-clone state.
func Status(poolRoot, project string) error {
	if _, err := os.Stat(pool.ProjectDir(poolRoot, project)); os.IsNotExist(err) {
		return fmt.Errorf("no pool for %q yet — run `git workpool setup`", project)
	}
	fmt.Printf("pool: %s\n", pool.ProjectDir(poolRoot, project))
	if _, err := os.Stat(pool.HubDir(poolRoot, project)); err == nil {
		if out, err := gitx.Run(pool.HubDir(poolRoot, project), "for-each-ref", "--format=%(refname:short)", "refs/heads"); err == nil && out != "" {
			fmt.Printf("hub branches: %s\n", strings.Join(strings.Split(out, "\n"), ", "))
		}
	} else {
		fmt.Println("hub: missing — run `git workpool setup`")
	}
	for _, s := range clone.States(poolRoot, project) {
		mark := "free"
		if s.Busy() {
			mark = "busy"
		}
		extra := ""
		if s.Dirty > 0 {
			extra += fmt.Sprintf(" %ddirty", s.Dirty)
		}
		if s.Ahead > 0 {
			extra += fmt.Sprintf(" %dahead", s.Ahead)
		}
		if s.Branch != s.Default && !s.HasHub {
			extra += " never-pushed"
		}
		fmt.Printf("  %-20s %-5s %s%s\n", s.Name, mark, s.Branch, extra)
	}
	return nil
}

// Claim syncs a free clone to branch; --force NAME rescues + resets it first.
func Claim(poolRoot, project, force, branch string) error {
	states := clone.States(poolRoot, project)
	if len(states) == 0 {
		return fmt.Errorf("no clones in pool — run `git workpool setup`")
	}
	target, err := clone.PickTarget(states, force)
	if err != nil {
		return err
	}
	if force != "" {
		if err := clone.Rescue(target); err != nil {
			return err
		}
	}
	// resolve branch: explicit, or re-engage the clone's existing branch
	if branch == "" {
		if target.Branch != target.Default && target.HasHub {
			branch = target.Branch
		} else {
			return fmt.Errorf("no branch given and clone is on %q — pass a branch", target.Branch)
		}
	}
	// sync: reset to hub's copy of the branch, or create from hub default
	if out, err := gitx.Run(target.Path, "fetch", "origin"); err != nil {
		return fmt.Errorf("fetch hub failed: %v\n%s", err, out)
	}
	if _, err := gitx.Run(target.Path, "rev-parse", "--verify", "--quiet", "origin/"+branch); err == nil {
		if out, err := gitx.Run(target.Path, "checkout", "-B", branch, "origin/"+branch); err != nil {
			return fmt.Errorf("checkout failed: %v\n%s", err, out)
		}
	} else {
		db := clone.DefaultBranch(target.Path)
		if _, err := gitx.Run(target.Path, "rev-parse", "--verify", "--quiet", "origin/"+db); err != nil {
			return fmt.Errorf("hub has no %q branch — publish it from your main clone first: `git workpool publish %s`", db, db)
		}
		if out, err := gitx.Run(target.Path, "checkout", "-b", branch, "origin/"+db); err != nil {
			return fmt.Errorf("checkout failed: %v\n%s", err, out)
		}
	}
	fmt.Printf("claimed clone %s at %s (branch %s)\n", target.Name, target.Path, branch)
	return nil
}

// Publish pushes the current branch to the hub. Never commits.
// In the main clone the branch exists only as a hub ref, so push HEAD:<branch>.
func Publish(info pool.Info, branch string) error {
	if branch == "" {
		branch, _ = gitx.Run(info.Root, "rev-parse", "--abbrev-ref", "HEAD")
	}
	if info.IsClone {
		if _, err := gitx.Run(info.Root, "remote", "get-url", "origin"); err != nil {
			return fmt.Errorf("no origin remote — run `git workpool setup` first")
		}
		out, err := gitx.Run(info.Root, "push", "origin", branch)
		if out != "" {
			fmt.Println(out)
		}
		if err != nil {
			return fmt.Errorf("publish failed: %v", err)
		}
		return nil
	}
	if _, err := gitx.Run(info.Root, "remote", "get-url", "hub"); err != nil {
		return fmt.Errorf("no hub remote — run `git workpool setup` first")
	}
	out, err := gitx.Run(info.Root, "push", "hub", "HEAD:"+branch)
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}
	return nil
}

// Pull fetches and merges the branch from the hub into the main clone.
func Pull(info pool.Info, branch string) error {
	if info.IsClone {
		return fmt.Errorf("pull runs in the main clone, not inside a workpool clone")
	}
	if _, err := gitx.Run(info.Root, "remote", "get-url", "hub"); err != nil {
		return fmt.Errorf("no hub remote — run `git workpool setup` first")
	}
	if branch == "" {
		branch, _ = gitx.Run(info.Root, "rev-parse", "--abbrev-ref", "HEAD")
	}
	if out, err := gitx.Run(info.Root, "fetch", "hub"); err != nil {
		return fmt.Errorf("fetch hub failed: %v\n%s", err, out)
	}
	out, err := gitx.Run(info.Root, "merge", "hub/"+branch)
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		return fmt.Errorf("merge failed — resolve conflicts, then commit: %v", err)
	}
	return nil
}

// Close discards local clone state and resets to the hub default branch.
func Close(poolRoot string, info pool.Info) error {
	if !info.IsClone {
		return fmt.Errorf("close runs inside a workpool clone")
	}
	states := clone.States(poolRoot, info.Project)
	var s *clone.State
	for i := range states {
		if states[i].Name == info.Clone {
			s = &states[i]
			break
		}
	}
	if s == nil {
		return fmt.Errorf("%s is not a workpool clone", info.Clone)
	}
	fmt.Printf("closing %s (on %s):", s.Name, s.Branch)
	switch {
	case s.Dirty > 0 && s.Ahead > 0:
		fmt.Printf(" %d change(s) and %d commit(s) will be discarded", s.Dirty, s.Ahead)
	case s.Dirty > 0:
		fmt.Printf(" %d change(s) will be discarded", s.Dirty)
	case s.Ahead > 0:
		fmt.Printf(" %d commit(s) will be discarded", s.Ahead)
	default:
		fmt.Print(" clean, nothing to discard")
	}
	fmt.Println()
	db := clone.DefaultBranch(s.Path)
	if out, err := gitx.Run(s.Path, "fetch", "origin"); err != nil {
		return fmt.Errorf("fetch failed: %v\n%s", err, out)
	}
	if out, err := gitx.Run(s.Path, "checkout", "-B", db, "origin/"+db); err != nil {
		return fmt.Errorf("reset failed: %v\n%s", err, out)
	}
	if out, err := gitx.Run(s.Path, "clean", "-fd"); err != nil {
		return fmt.Errorf("clean failed: %v\n%s", err, out)
	}
	fmt.Printf("clone %s free\n", s.Name)
	return nil
}

// ParseClaimArgs splits claim flags: --force NAME, plus one branch argument.
func ParseClaimArgs(args []string) (force, branch string, err error) {
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

// BranchArg returns the first non-flag argument, if any.
func BranchArg(args []string) string {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0]
	}
	return ""
}

// installDeps runs the project's package manager install in the clone.
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
