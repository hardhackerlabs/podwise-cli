package podcast

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// Unfollow unfollows the podcast identified by seq. The operation is idempotent —
// unfollowing a podcast you do not follow succeeds silently.
func Unfollow(ctx context.Context, client *api.Client, seq int) error {
	path := fmt.Sprintf("/open/v1/podcasts/%d/unfollow", seq)
	var resp struct {
		Success bool `json:"success"`
	}
	if err := client.Post(ctx, path, nil, &resp); err != nil {
		return formatUnfollowError(err)
	}
	return nil
}

// formatUnfollowError translates API errors into user-friendly messages.
func formatUnfollowError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "not_found":
		return fmt.Errorf("podcast does not exist")
	default:
		return err
	}
}
