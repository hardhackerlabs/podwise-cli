package episode

import (
	"context"
	"fmt"
	"strings"

	"github.com/hardhacker/podwise-cli/internal/api"
)

// TranslationStatus represents the processing state of a single translation.
// Status is nil when the translation has not been started.
type TranslationStatus struct {
	Status   *string `json:"status"`
	Progress int     `json:"progress"`
}

// TranslationMap maps language names (e.g. "Chinese", "Japanese") to their
// current translation status.
type TranslationMap map[string]TranslationStatus

type translateRequest struct {
	Language string `json:"language"`
}

type translateResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Message string `json:"message"`
	} `json:"result"`
}

type listTranslationsResponse struct {
	Success bool           `json:"success"`
	Result  TranslationMap `json:"result"`
}

// RequestTranslation submits a translation request for the given episode seq into
// the specified target language. If the translation already exists and is complete,
// the request is a no-op.
//
// language must be one of the API-accepted values: "Chinese", "Traditional Chinese",
// "English", "Japanese", "Korean".
//
// Returns an error if:
//   - language is missing or invalid (400)
//   - a paid plan is required (402)
//   - the episode does not exist (404 not_found)
//   - the API request fails for other reasons
func RequestTranslation(ctx context.Context, client *api.Client, seq int, language string) error {
	language = strings.ReplaceAll(language, "-", " ")
	path := fmt.Sprintf("/open/v1/episodes/%d/translate", seq)

	var resp translateResponse
	if err := client.Post(ctx, path, translateRequest{Language: language}, &resp); err != nil {
		return formatRequestTranslationError(err)
	}

	if !resp.Success {
		return fmt.Errorf("request translation for episode %d: operation failed", seq)
	}

	return nil
}

// ListTranslations retrieves the available translations and their processing status
// for the given episode seq.
//
// The returned TranslationMap is keyed by language name. A nil Status field
// indicates the translation has not been started yet.
//
// Returns an error if the episode does not exist or the API request fails.
func ListTranslations(ctx context.Context, client *api.Client, seq int) (TranslationMap, error) {
	path := fmt.Sprintf("/open/v1/episodes/%d/translations", seq)

	var resp listTranslationsResponse
	if err := client.Get(ctx, path, nil, &resp); err != nil {
		return nil, formatListTranslationsError(err)
	}

	return resp.Result, nil
}

// formatRequestTranslationError translates API errors from the translate endpoint
// into user-friendly messages.
func formatRequestTranslationError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch {
	case apiErr.StatusCode == 400:
		return fmt.Errorf("invalid language: must be one of %s", strings.Join(LanguageNames(), ", "))
	case apiErr.StatusCode == 402:
		return fmt.Errorf("paid plan required: translation feature is not available on your current plan")
	case apiErr.ErrCode == "not_found":
		return fmt.Errorf("episode does not exist")
	default:
		return err
	}
}

// formatListTranslationsError translates API errors from the list translations
// endpoint into user-friendly messages.
func formatListTranslationsError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok {
		return err
	}

	switch apiErr.ErrCode {
	case "not_found":
		return fmt.Errorf("episode does not exist")
	default:
		return err
	}
}
