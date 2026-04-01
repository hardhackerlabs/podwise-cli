package episode

import (
	"context"
	"fmt"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/async"
)

// NotionExportOptions holds parameters for exporting to Notion.
type NotionExportOptions struct {
	// Transcripts controls whether to include transcript content.
	Transcripts bool
	// Mindmap controls whether to include the mind map.
	Mindmap bool
	// Translation is the target language code (e.g., "zh", "ja").
	// Empty string means no translation.
	Translation string
	// MixWithOriginLanguage controls whether to show both original and translated text.
	MixWithOriginLanguage bool
}

// NotionExportResult holds the response from the Notion export API.
type NotionExportResult struct {
	URL     string `json:"url"`
	Warning string `json:"warning,omitempty"`
}

type notionExportResponse struct {
	Success bool               `json:"success"`
	Result  NotionExportResult `json:"result"`
}

// ExportToNotion sends an episode's content to the user's connected Notion workspace.
// Returns the URL of the created Notion page and an optional warning message.
//
// Common error codes:
//   - not_connected: Notion is not connected
//   - not_configured: Notion database is not configured
//   - property_not_exists: Required property missing from Notion database
//   - unauthorized: Notion API token is invalid or expired
//   - database_not_found: Notion database not found
//   - rate_limited: Notion API rate limited
//   - notion_error: Unexpected Notion API error
//   - timeout: Notion API request timed out
func ExportToNotion(ctx context.Context, client *api.Client, seq int, opts NotionExportOptions) (*NotionExportResult, error) {
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/send/notion", seq)

	// Build request body with only non-default values
	body := make(map[string]any)

	// Only include fields that differ from defaults
	if !opts.Transcripts {
		body["transcripts"] = false
	}
	if opts.Mindmap {
		body["mindmap"] = true
	}
	if opts.Translation != "" {
		body["translation"] = opts.Translation
	}
	if opts.MixWithOriginLanguage {
		body["mixWithOriginLanguage"] = true
	}
	body["mixOutlines"] = false

	var resp notionExportResponse
	if err := client.Post(ctx, apiPath, body, &resp); err != nil {
		return nil, formatNotionError(err)
	}

	// Mark episode as read in background
	async.Go(func() {
		_ = MarkAsRead(context.Background(), client, seq)
	})

	return &resp.Result, nil
}

// formatNotionError translates API errors into user-friendly messages.
func formatNotionError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "not_connected":
		return fmt.Errorf("notion is not connected: connect via Podwise settings at https://podwise.ai/dashboard/settings")
	case "not_configured":
		return fmt.Errorf("notion database is not configured: configure via Podwise settings at https://podwise.ai/dashboard/settings")
	case "property_not_exists":
		return fmt.Errorf("required property missing from notion database: %s", apiErr.Message)
	case "unauthorized":
		return fmt.Errorf("notion authentication failed: reconnect via Podwise settings at https://podwise.ai/dashboard/settings")
	case "database_not_found":
		return fmt.Errorf("notion database not found: reconfigure via Podwise settings at https://podwise.ai/dashboard/settings")
	case "rate_limited":
		return fmt.Errorf("notion API rate limit exceeded: please try again later")
	case "notion_error":
		return fmt.Errorf("notion API error: %s", apiErr.Message)
	case "timeout":
		return fmt.Errorf("notion API request timed out: please try again")
	default:
		return err
	}
}

// ReadwiseExportOptions holds parameters for exporting to Readwise Reader.
type ReadwiseExportOptions struct {
	// Mindmap controls whether to include the mind map as nested list.
	Mindmap bool
	// Shownotes controls whether to include episode shownotes.
	Shownotes bool
	// Location specifies where to save in Reader: "new" (inbox), "later", or "archive".
	Location string
	// Translation is the target language code (e.g., "zh", "ja").
	// Empty string means no translation.
	Translation string
	// MixWithOriginLanguage controls whether to show both original and translated text.
	MixWithOriginLanguage bool
}

// ReadwiseExportResult holds the response from the Readwise export API.
type ReadwiseExportResult struct {
	URL string `json:"url"`
}

type readwiseExportResponse struct {
	Success bool                 `json:"success"`
	Result  ReadwiseExportResult `json:"result"`
}

// ExportToReadwise sends an episode's content to the user's connected Readwise Reader account.
// Returns the URL of the created Reader document.
//
// Common error codes:
//   - not_connected: Readwise is not connected
//   - unauthorized: Readwise API token is invalid or expired
//   - readwise_error: Unexpected Readwise API error
func ExportToReadwise(ctx context.Context, client *api.Client, seq int, opts ReadwiseExportOptions) (*ReadwiseExportResult, error) {
	apiPath := fmt.Sprintf("/open/v1/episodes/%d/send/reader", seq)

	// Build request body with only non-default values
	body := make(map[string]any)

	if opts.Mindmap {
		body["mindmap"] = true
	}
	if opts.Shownotes {
		body["shownotes"] = true
	}
	if opts.Location != "" && opts.Location != "archive" {
		body["location"] = opts.Location
	}
	if opts.Translation != "" {
		body["translation"] = opts.Translation
	}
	if opts.MixWithOriginLanguage {
		body["mixWithOriginLanguage"] = true
	}
	body["mixOutlines"] = false

	var resp readwiseExportResponse
	if err := client.Post(ctx, apiPath, body, &resp); err != nil {
		return nil, formatReadwiseError(err)
	}

	// Mark episode as read in background
	async.Go(func() {
		_ = MarkAsRead(context.Background(), client, seq)
	})

	return &resp.Result, nil
}

// formatReadwiseError translates API errors into user-friendly messages.
func formatReadwiseError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "not_connected":
		return fmt.Errorf("readwise is not connected: configure API token via Podwise settings at https://podwise.ai/dashboard/settings")
	case "unauthorized":
		return fmt.Errorf("readwise authentication failed: reconfigure API token via Podwise settings at https://podwise.ai/dashboard/settings")
	case "readwise_error":
		return fmt.Errorf("readwise API error: %s", apiErr.Message)
	default:
		return err
	}
}
