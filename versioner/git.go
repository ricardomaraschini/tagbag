package versioner

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Git provides sort of a versioner for the images using git as
// backend. It allows to commit and tag the current state of the
// images (in something we conventionally call a "snapshot").
type Git struct {
	basedir string
	repo    *git.Repository
	mtx     sync.Mutex
}

// New returns a reference to a Git using provided directory as
// base for the repository.
func NewGit(basedir string) (*Git, error) {
	repo, err := git.PlainInit(basedir, false)
	if err != nil {
		repo, err = git.PlainOpen(basedir)
		if err != nil {
			return nil, err
		}
	}
	return &Git{basedir: basedir, repo: repo}, nil
}

// Commit commits the current state of the repository. Creates a
// commit with a default message and the current time.
func (g *Git) Commit(message string) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	wt, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}
	if _, err := wt.Add("."); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}
	opts := &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Tag Bag System",
			Email: "tagbag@example.com",
			When:  time.Now(),
		},
	}
	if _, err := wt.Commit(message, opts); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// Snapshot creates a tag for the current state of the repository.
func (g *Git) Snapshot(name string) error {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	ref, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get head: %w", err)
	}
	if _, err = g.repo.CreateTag(name, ref.Hash(), nil); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	return nil
}
