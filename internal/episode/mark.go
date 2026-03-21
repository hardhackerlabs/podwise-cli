package episode

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// MarkResponse represents the API response for mark read/unread operations.
type MarkResponse struct {
	Success bool `json:"success"`
}

// MarkAsRead marks an episode as read for the authenticated user.
// The operation is idempotent - if the episode is already marked as read,
// the request succeeds silently.
//
// Returns an error if:
//   - The episode does not exist (404 not_found)
//   - The API request fails for other reasons
func MarkAsRead(ctx context.Context, client *api.Client, seq int) error {
	path := fmt.Sprintf("/open/v1/episodes/%d/read", seq)

	var resp MarkResponse
	if err := client.Post(ctx, path, nil, &resp); err != nil {
		return fmt.Errorf("mark episode %d as read: %w", seq, err)
	}

	if !resp.Success {
		return fmt.Errorf("mark episode %d as read: operation failed", seq)
	}

	return nil
}

// MarkAsUnread marks an episode as unread for the authenticated user.
// The operation is idempotent - if the episode is already unread or has no
// read record, the request succeeds silently.
//
// Returns an error if:
//   - The episode does not exist (404 not_found)
//   - The API request fails for other reasons
func MarkAsUnread(ctx context.Context, client *api.Client, seq int) error {
	path := fmt.Sprintf("/open/v1/episodes/%d/unread", seq)

	var resp MarkResponse
	if err := client.Post(ctx, path, nil, &resp); err != nil {
		return fmt.Errorf("mark episode %d as unread: %w", seq, err)
	}

	if !resp.Success {
		return fmt.Errorf("mark episode %d as unread: operation failed", seq)
	}

	return nil
}
