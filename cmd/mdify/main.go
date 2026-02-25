package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"mdify/internal/filesystem"
	"mdify/pkg/scraper"
	"mdify/pkg/server"
	"mdify/pkg/sitemap"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "mdify",
		Short:   "Convert web documentation to markdown files",
		Long:    `Scrape docs sites and convert them to markdown for LLM consumption, preserving directory structure.`,
		Version: "0.1.0",
	}

	rootCmd.AddCommand(scrapeCmd())
	rootCmd.AddCommand(serveCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scrapeCmd() *cobra.Command {
	var (
		selector   string
		output     string
		sitemapURL string
		pathFilter string
		workers    int
	)

	cmd := &cobra.Command{
		Use:   "scrape [urls-file]",
		Short: "Scrape URLs and convert to markdown",
		Long: `Scrape web pages from URLs and convert to markdown files.

Examples:
  # From file or stdin
  mdify scrape --selector ".content" urls.txt
  cat urls.txt | mdify scrape --selector ".content"

  # From sitemap
  mdify scrape --sitemap https://example.com/sitemap.xml --selector ".content"

  # From sitemap with path filtering
  mdify scrape --sitemap https://example.com/sitemap.xml --filter "/docs/" --selector ".prose"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var urls []string
			var err error

			if sitemapURL != "" {
				if len(args) > 0 {
					return fmt.Errorf("cannot use both sitemap and URL file")
				}
				urls, err = getURLsFromSitemap(sitemapURL, pathFilter)
				if err != nil {
					return fmt.Errorf("failed to get URLs from sitemap: %w", err)
				}
			} else {
				if len(args) == 0 {
					urls, err = readURLsFromStdin()
				} else {
					urls, err = readURLsFromFile(args[0])
				}
				if err != nil {
					return fmt.Errorf("failed to read URLs: %w", err)
				}
			}

			if len(urls) == 0 {
				return fmt.Errorf("no URLs found to scrape")
			}

			return runScrapeCommand(urls, selector, output, workers)
		},
	}

	cmd.Flags().StringVarP(&selector, "selector", "s", "", "CSS selector for content extraction (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "./docs", "Output directory for markdown files")
	cmd.Flags().StringVar(&sitemapURL, "sitemap", "", "URL to sitemap.xml file")
	cmd.Flags().StringVar(&pathFilter, "filter", "", "Filter URLs containing this path (e.g. '/docs/')")
	cmd.Flags().IntVarP(&workers, "workers", "w", 4, "Number of concurrent workers (default: 4, use 1 for sequential)")
	cmd.MarkFlagRequired("selector")

	return cmd
}

func serveCmd() *cobra.Command {
	var (
		dir  string
		port int
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve markdown files via HTTP",
		Long:  `Start an HTTP server to serve the converted markdown files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServeCommand(dir, port)
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "d", "./docs", "Directory containing markdown files")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to serve on")

	return cmd
}

func readURLsFromStdin() ([]string, error) {
	var urls []string
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			urls = append(urls, line)
		}
	}

	return urls, scanner.Err()
}

func readURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			urls = append(urls, line)
		}
	}

	return urls, scanner.Err()
}

type RealSleeper struct{}

func (s RealSleeper) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

type RealLogger struct{}

func (l RealLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

func runScrapeCommand(urls []string, selector, output string, workers int) error {
	client := &http.Client{Timeout: 30 * time.Second}
	fs := filesystem.OSFileSystem{}
	sleeper := RealSleeper{}
	logger := RealLogger{}
	config := scraper.Config{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		Workers:    workers,
	}

	service := scraper.NewService(client, fs, sleeper, logger, config)
	return service.ScrapeURLs(urls, selector, output)
}

func getURLsFromSitemap(sitemapURL, pathFilter string) ([]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	logger := RealLogger{}
	service := sitemap.NewService(client, logger)
	return service.GetURLsFromSitemap(sitemapURL, pathFilter)
}

func runServeCommand(dir string, port int) error {
	fs := filesystem.OSFileSystem{}
	logger := RealLogger{}
	service := server.NewService(fs, logger)
	return service.ServeMarkdownFiles(dir, port)
}
