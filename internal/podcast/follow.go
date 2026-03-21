package podcast

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// Follow follows the podcast identified by seq. The operation is idempotent —
// following an already-followed podcast succeeds silently.
func Follow(ctx context.Context, client *api.Client, seq int) error {
	path := fmt.Sprintf("/open/v1/podcasts/%d/follow", seq)
	var resp struct {
		Success bool `json:"success"`
	}
	if err := client.Post(ctx, path, nil, &resp); err != nil {
		return formatFollowError(err)
	}
	return nil
}

// formatFollowError translates API errors into user-friendly messages.
func formatFollowError(err error) error {
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
