package main

import (
	"fmt"
	"net/url"
	"runtime"

	"github.com/spf13/cobra"
)

// feedbackCmd opens a prefilled GitHub Discussion so users can report what's
// working and what isn't. It does not auto-submit anything; the user reviews
// the prefilled URL and posts manually. See docs/north-star.md.
var feedbackCmd = &cobra.Command{
	Use:   "feedback",
	Short: "Open a prefilled feedback discussion on GitHub",
	Long: `Open a GitHub Discussion prefilled with your platform info and Preflight
version so the maintainers can get useful feedback without you having to type
it all out. Nothing is sent automatically — you review the URL first.`,
	RunE: runFeedback,
}

func init() {
	rootCmd.AddCommand(feedbackCmd)
}

func runFeedback(_ *cobra.Command, _ []string) error {
	u := buildFeedbackURL(runtime.GOOS, runtime.GOARCH, version)

	fmt.Println("Open this URL in your browser to share feedback:")
	fmt.Println()
	fmt.Println("  " + u)
	fmt.Println()
	fmt.Println("Nothing is sent until you click Submit on the GitHub page.")
	return nil
}

// buildFeedbackURL returns a URL that opens GitHub Discussions with the body
// prefilled. Exposed for testing — callers in production go through
// runFeedback.
func buildFeedbackURL(goos, goarch, ver string) string {
	body := fmt.Sprintf(`### What were you trying to do?

<!-- e.g. "set up a new MacBook from my dotfiles repo" -->

### What worked, what did not

<!-- the more concrete the better -->

### Environment

- OS: %s
- Arch: %s
- Preflight version: %s

<!-- Anything else helpful goes below this line -->
`, goos, goarch, ver)

	q := url.Values{}
	q.Set("category", "general")
	q.Set("title", "Feedback: ")
	q.Set("body", body)

	return "https://github.com/felixgeelhaar/preflight/discussions/new?" + q.Encode()
}
