package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v47/github"
	"github.com/whywaita/myshoes/pkg/datastore"
	"github.com/whywaita/myshoes/pkg/gh"
	"github.com/whywaita/myshoes/pkg/logger"
)

// removeRunnerModeEphemeral remove runner that created by --ephemeral flag.
// --ephemeral flag is delete self-hosted runner when end of job. So, The origin list of runner from datastore.
func (m *Manager) removeRunnerModeEphemeral(ctx context.Context, t datastore.Target, runner datastore.Runner, ghRunners []*github.Runner) error {
	owner, repo := t.OwnerRepo()
	client, err := gh.NewClient(t.GitHubToken)
	if err != nil {
		return fmt.Errorf("failed to create github client: %w", err)
	}

	ghRunner, err := gh.ExistGitHubRunnerWithRunner(ghRunners, ToName(runner.UUID.String()))
	logger.Logf(false, "check the runner for deletion: %v", ghRunner)
	switch {
	case errors.Is(err, gh.ErrNotFound):
		// deleted in GitHub, It's completed
		if err := m.deleteRunner(ctx, runner, StatusWillDelete); err != nil {
			if err := datastore.UpdateTargetStatus(ctx, m.ds, t.UUID, datastore.TargetStatusErr, ""); err != nil {
				logger.Logf(false, "failed to update target status (target ID: %s): %+v\n", t.UUID, err)
			}

			return fmt.Errorf("failed to delete runner: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("failed to check runner exist in GitHub (runner: %s): %w", runner.UUID, err)
	}

	if err := sanitizeGitHubRunner(*ghRunner, runner); err != nil {
		if errors.Is(err, ErrNotWillDeleteRunner) {
			return nil
		}

		return fmt.Errorf("failed to check runner of status: %w", err)
	}

	if err := m.deleteRunnerWithGitHub(ctx, client, runner, ghRunner.GetID(), owner, repo, ghRunner.GetStatus()); err != nil {
		if err := datastore.UpdateTargetStatus(ctx, m.ds, t.UUID, datastore.TargetStatusErr, ""); err != nil {
			logger.Logf(false, "failed to update target status (target ID: %s): %+v\n", t.UUID, err)
		}

		return fmt.Errorf("failed to delete runner with GitHub: %w", err)
	}
	return nil
}
