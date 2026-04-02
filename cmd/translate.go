package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hardhacker/podwise-cli/internal/api"
	"github.com/hardhacker/podwise-cli/internal/config"
	"github.com/hardhacker/podwise-cli/internal/episode"
	"github.com/spf13/cobra"
)

var translateLang string
var translatePollInterval time.Duration
var translateTimeout time.Duration

// podwise translate <episode-url>
var translateCmd = &cobra.Command{
	Use:   "translate <episode-url>",
	Short: "Translate an episode's transcript and summary into another language",
	Long: `Request translation of an episode's transcript and summary into a target language,
then wait until the translation is complete.

If the translation already exists and is complete, the request is a no-op.

Supported languages: ` + strings.Join(episode.LanguageNames(), ", "),
	Example: `  podwise translate https://podwise.ai/dashboard/episodes/7360326 --lang Chinese
  podwise translate https://podwise.ai/dashboard/episodes/7360326 --lang Japanese`,
	Args: cobra.ExactArgs(1),
	RunE: runTranslate,
}

func init() {
	langUsage := "target language: " + strings.Join(episode.LanguageNames(), ", ")
	translateCmd.Flags().StringVar(&translateLang, "lang", "", langUsage)
	_ = translateCmd.MarkFlagRequired("lang")
	translateCmd.Flags().DurationVar(&translatePollInterval, "interval", 10*time.Second, "how often to poll for translation status")
	translateCmd.Flags().DurationVar(&translateTimeout, "timeout", 5*time.Minute, "maximum time to wait for translation to complete")
}

func runTranslate(cmd *cobra.Command, args []string) error {
	seq, err := episode.ParseSeq(args[0])
	if err != nil {
		return fmt.Errorf("invalid episode: %w", err)
	}

	langName, err := episode.ResolveLangName(translateLang)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	client := api.New(cfg.APIBaseURL, cfg.APIKey)
	ctx := context.Background()

	fmt.Printf("Requesting %s translation for episode %s ...\n", langName, episode.BuildEpisodeURL(seq))

	if err := episode.RequestTranslation(ctx, client, seq, langName); err != nil {
		return err
	}

	fmt.Printf("Translation request submitted. Waiting for completion...\n\n")

	// Check immediately before the first tick.
	if done, err := checkTranslationStatus(ctx, client, seq, langName); err != nil {
		return err
	} else if done {
		printTranslateDoneHint(seq, langName)
		return nil
	}

	deadline := time.Now().Add(translateTimeout)
	ticker := time.NewTicker(translatePollInterval)
	defer ticker.Stop()

	for range ticker.C {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for %s translation of episode %s",
				translateTimeout, langName, episode.BuildEpisodeURL(seq))
		}
		done, err := checkTranslationStatus(ctx, client, seq, langName)
		if err != nil {
			return err
		}
		if done {
			printTranslateDoneHint(seq, langName)
			return nil
		}
	}

	return nil
}

// checkTranslationStatus polls ListTranslations once and prints the current
// status line. Returns (true, nil) when the translation is done, (false, nil)
// while still in progress, and (false, err) when the translation has failed or
// the API call itself failed.
func checkTranslationStatus(ctx context.Context, client *api.Client, seq int, langName string) (bool, error) {
	translations, err := episode.ListTranslations(ctx, client, seq)
	if err != nil {
		return false, err
	}

	ts := time.Now().Format("15:04:05")
	t, ok := translations[langName]
	if !ok || t.Status == nil {
		fmt.Printf("  [%s] → pending      translation not started yet\n", ts)
		return false, nil
	}

	switch *t.Status {
	case "done":
		fmt.Printf("  [%s] ✓ done         translation complete (100%%)\n", ts)
		return true, nil
	case "failed":
		return false, fmt.Errorf("translation failed for episode %s", episode.BuildEpisodeURL(seq))
	default:
		// "processing" and any other transitional status.
		fmt.Printf("  [%s] → processing   %d%% complete\n", ts, t.Progress)
		return false, nil
	}
}

func printTranslateDoneHint(seq int, langName string) {
	episodeURL := episode.BuildEpisodeURL(seq)
	sep := "─────────────────────────────────────────────────────────"
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  ✓  Translation Complete (%s)\n", langName)
	fmt.Printf("%s\n", sep)
	fmt.Printf("  Episode URL:   %s\n", episodeURL)
	fmt.Printf("\n")
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    podwise get summary    --lang %s %s\n", langName, episodeURL)
	fmt.Printf("    podwise get transcript --lang %s %s\n", langName, episodeURL)
}
