// Package command implements the git workpool subcommands.
package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/avrebarra/git-workpool/internal/clone"
	"github.com/avrebarra/git-workpool/internal/gitx"
	"github.com/avrebarra/git-workpool/internal/pool"
	"github.com/avrebarra/git-workpool/internal/util"
)

// Setup creates the hub (first run) and one codenamed clone per call.
func Setup(poolRoot string, info pool.Info) error {
	if info.IsClone {
		return fmt.Errorf("setup runs in the main clone, not inside a workpool clone")
	}

	// create the bare hub once per project
	hub := pool.HubDir(poolRoot, info.Project)
	if !util.FileExists(pool.ProjectDir(poolRoot, info.Project), "hub.git") {
		if err := os.MkdirAll(pool.ProjectDir(poolRoot, info.Project), 0o755); err != nil {
			return err
		}
		if err := gitx.CloneBare(info.Root, hub); err != nil {
			return fmt.Errorf("create hub: %v", err)
		}
		fmt.Printf("hub created: %s\n", hub)
	} else {
		fmt.Printf("hub exists: %s\n", hub)
	}

	// wire the main clone to the hub
	if !gitx.HasRemote(info.Root, "hub") {
		if err := gitx.AddRemote(info.Root, "hub", hub); err != nil {
			return fmt.Errorf("add hub remote: %v", err)
		}
		fmt.Printf("hub remote added\n")
	}

	// create one codenamed clone per call
	name := clone.NextCodename(poolRoot, info.Project)
	clonePath := filepath.Join(pool.ProjectDir(poolRoot, info.Project), name)
	if util.FileExists(pool.ProjectDir(poolRoot, info.Project), name) {
		return fmt.Errorf("clone %q already exists at %s", name, clonePath)
	}
	if err := gitx.Clone(hub, clonePath); err != nil {
		return fmt.Errorf("create clone: %v", err)
	}
	installDeps(clonePath)
	fmt.Printf("clone created: %s (%s)\n", name, clonePath)
	return nil
}

// Status prints hub branches and per-clone state.
func Status(poolRoot, project string) error {
	if !util.FileExists(poolRoot, project) {
		return fmt.Errorf("no pool for %q yet — run `git workpool setup`", project)
	}
	fmt.Printf("pool: %s\n", pool.ProjectDir(poolRoot, project))

	// hub branches
	if util.FileExists(pool.ProjectDir(poolRoot, project), "hub.git") {
		if branches, err := gitx.ListBranches(pool.HubDir(poolRoot, project)); err == nil && len(branches) > 0 {
			fmt.Printf("hub branches: %s\n", joinNames(branches))
		}
	} else {
		fmt.Println("hub: missing — run `git workpool setup`")
	}

	// per-clone state
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

// Claim syncs a free clone to branch; --clone NAME pins to a clone, --force rescues + resets it first.
func Claim(poolRoot, project, cloneName string, force bool, branch string) error {
	states := clone.States(poolRoot, project)
	if len(states) == 0 {
		return fmt.Errorf("no clones in pool — run `git workpool setup`")
	}
	target, err := clone.PickTarget(states, cloneName, force)
	if err != nil {
		return err
	}

	// force-claims rescue the clone's work before touching it
	if force && cloneName != "" {
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

	// sync the clone: reset to the hub's copy, or branch off the default
	if err := gitx.Fetch(target.Path, "origin"); err != nil {
		return fmt.Errorf("fetch hub failed: %v", err)
	}
	if gitx.HasRemoteBranch(target.Path, branch) {
		if err := gitx.CheckoutBranch(target.Path, branch); err != nil {
			return fmt.Errorf("checkout failed: %v", err)
		}
	} else {
		if err := gitx.CheckoutNewBranch(target.Path, branch, gitx.GetDefaultBranch(target.Path)); err != nil {
			return fmt.Errorf("checkout failed: %v", err)
		}
	}
	fmt.Printf("claimed clone %s at %s (branch %s)\n", target.Name, target.Path, branch)
	return nil
}

// Store sends committed work to the local hub. Never commits, never touches
// the remote. From a clone it pushes origin (the hub). From the main clone it
// pushes HEAD when you are on the branch, else finds the clone working on it.
func Store(poolRoot string, info pool.Info, branch string) error {
	if branch == "" {
		branch, _ = gitx.GetCurrentBranch(info.Root)
	}
	if info.IsClone {
		return storeFromClone(info, branch)
	}
	return storeFromMain(poolRoot, info, branch)
}

// storeFromClone pushes the clone's branch to its origin (the hub).
func storeFromClone(info pool.Info, branch string) error {
	if !gitx.HasRemote(info.Root, "origin") {
		return fmt.Errorf("no origin remote — run `git workpool setup` first")
	}
	out, err := gitx.PushBranch(info.Root, "origin", branch)
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		return fmt.Errorf("store failed: %v", err)
	}
	return nil
}

// storeFromMain pushes to the hub remote. On the branch (review edits) it
// pushes HEAD; otherwise it finds the clone working on the branch and pushes
// from there. Only ever talks to the local hub — never the real remote.
func storeFromMain(poolRoot string, info pool.Info, branch string) error {
	if !gitx.HasRemote(info.Root, "hub") {
		return fmt.Errorf("no hub remote — run `git workpool setup` first")
	}
	cur, _ := gitx.GetCurrentBranch(info.Root)
	if cur == branch {
		out, err := gitx.PushHeadBranch(info.Root, "hub", branch)
		if out != "" {
			fmt.Println(out)
		}
		if err != nil {
			return fmt.Errorf("store failed: %v", err)
		}
		return nil
	}
	for _, st := range clone.States(poolRoot, info.Project) {
		if st.Branch == branch {
			out, err := gitx.PushBranch(st.Path, "origin", branch)
			if out != "" {
				fmt.Println(out)
			}
			if err != nil {
				return fmt.Errorf("store failed: %v", err)
			}
			fmt.Printf("stored from clone %s\n", st.Name)
			return nil
		}
	}
	return fmt.Errorf("no clone is on branch %q and you are not on it — switch to %q in the main clone, or check `git workpool status`", branch, branch)
}

// HubFetch makes a hub branch available in the main clone as a local branch.
// No merge, no checkout — you switch to it yourself to review and test.
func HubFetch(info pool.Info, branch string) error {
	if info.IsClone {
		return fmt.Errorf("hub fetch runs in the main clone, not inside a workpool clone")
	}
	if !gitx.HasRemote(info.Root, "hub") {
		return fmt.Errorf("no hub remote — run `git workpool setup` first")
	}
	if branch == "" {
		branch, _ = gitx.GetCurrentBranch(info.Root)
	}
	if err := gitx.FetchBranch(info.Root, "hub", branch); err != nil {
		return fmt.Errorf("fetch hub failed: %v", err)
	}
	if !gitx.HasRemoteBranchOf(info.Root, "hub", branch) {
		return fmt.Errorf("hub has no branch %q — check `git workpool status`", branch)
	}
	if !gitx.HasLocalBranch(info.Root, branch) {
		if err := gitx.CreateBranch(info.Root, branch, "hub/"+branch); err != nil {
			return fmt.Errorf("create branch failed: %v", err)
		}
	}
	fmt.Printf("branch %s available in the main clone — switch with: git switch %s\n", branch, branch)
	return nil
}

// Close discards local clone state and resets to the hub default branch.
// Runs from anywhere: from the main clone pass the clone name, from inside a
// clone the name defaults to the clone you're in.
func Close(poolRoot string, info pool.Info, name string) error {
	if name == "" {
		if !info.IsClone {
			return fmt.Errorf("close needs a clone name — run `git workpool status` to see clones")
		}
		name = info.Clone
	}
	var s *clone.State
	for _, st := range clone.States(poolRoot, info.Project) {
		if st.Name == name {
			s = &st
			break
		}
	}
	if s == nil {
		return fmt.Errorf("no clone named %q in the pool — run `git workpool status`", name)
	}

	// report exactly what will be discarded before resetting
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

	// reset to the hub default branch and drop untracked files
	db := gitx.GetDefaultBranch(s.Path)
	if err := gitx.Fetch(s.Path, "origin"); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}
	if err := gitx.CheckoutBranch(s.Path, db); err != nil {
		return fmt.Errorf("reset failed: %v", err)
	}
	if err := gitx.Clean(s.Path); err != nil {
		return fmt.Errorf("clean failed: %v", err)
	}
	fmt.Printf("clone %s free\n", s.Name)
	return nil
}

// installDeps runs the project's package manager install in the clone.
func installDeps(path string) {
	pm := ""
	switch {
	case util.FileExists(path, "bun.lock"), util.FileExists(path, "bun.lockb"):
		pm = "bun"
	case util.FileExists(path, "pnpm-lock.yaml"):
		pm = "pnpm"
	case util.FileExists(path, "package-lock.json"):
		pm = "npm"
	case util.FileExists(path, "yarn.lock"):
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

// joinNames renders a branch list as a comma-separated line.
func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
