package versioner

import (
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
}

// New returns a reference to a Git using provided directory as
// base for the repository.
func NewGit(basedir string) (*Git, error) {
	repo, err := git.PlainInit(basedir, false)
	if err != nil {
		return nil, err
	}
	return &Git{basedir, repo}, nil
}

// Commit commits the current state of the repository. Creates a
// commit with a default message and the current time.
func (g *Git) Commit() error {
	wt, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	if _, err := wt.Add("."); err != nil {
		return err
	}
	_, err = wt.Commit(
		"auto commit",
		&git.CommitOptions{
			Author: &object.Signature{
				Name:  "Tag Bag System",
				Email: "tagbag@example.com",
				When:  time.Now(),
			},
		},
	)
	return err
}

// Snapshot creates a tag for the current state of the repository.
func (g *Git) Snapshot(name string) error {
	ref, err := g.repo.Head()
	if err != nil {
		return err
	}
	_, err = g.repo.CreateTag(name, ref.Hash(), nil)
	if err != nil {
		return err
	}
	return nil
}
