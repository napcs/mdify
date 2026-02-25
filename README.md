# mdify

Convert web documentation sites to markdown files for LLM consumption, preserving directory structure.

You can use the resulting files locally, serve them as alternative content, or even point an MCP server at the output and serve it that way.

## Features

* **Sitemap support** - Automatically discover URLs from sitemap.xml files
* **URL filtering** - Filter URLs by path (e.g., only `/docs/` pages)
* **CSS selector extraction** - Extract specific content using CSS selectors
* **Concurrent processing** - Use multiple workers for faster scraping
* **Directory structure preservation** - Maintains original URL paths as file paths
* **Built-in HTTP server** - Serve converted markdown files for easy browsing
* **Retry logic** - Automatic retry with exponential backoff for failed requests

## Installation

Download the binary for your OS from the [releases page](https://github.com/napcs/mdify/releases) and place it on your `PATH`.

Or build from source:

```bash
git clone https://github.com/mdify/mdify.git
cd mdify
go build -o mdify cmd/mdify/main.go
```

## Usage

To use the tool, provide a list of URLS and the CSS selector that tells this tool where your content starts:

```bash
echo "https://example.com/docs" > urls.txt
mdify scrape --selector ".content" urls.txt
```

You can also provide the URLs from standard input:

```bash
echo "https://example.com/docs" | mdify scrape --selector ".content"
```

### Sitemap Usage

If your site has a valid sitemap, you can use that as the input.

Scrape all URLs from a sitemap:

```bash
mdify scrape --sitemap https://example.com/sitemap.xml --selector ".content"
```

If you only want a subset of files from the sitemap, you can filter the URLs by path:

```bash
mdify scrape --sitemap https://example.com/sitemap.xml --filter "/docs/" --selector ".prose"
```

### Concurrent Processing

The app uses multiple workers for faster processing of the files. The default is 4 workers. Use `--workers` to change the default:

```bash
mdify scrape --sitemap https://example.com/sitemap.xml --selector ".content" --workers 8
```

### Serve Converted Files

Start an HTTP server to browse the converted markdown files:

```bash
mdify serve --dir ./docs --port 8080
```

Then visit http://localhost:8080 to browse your converted documentation.

## Options

### Scrape Command

```
mdify scrape [urls-file]

Flags:
  -s, --selector string    CSS selector for content extraction (required)
  -o, --output string      Output directory for markdown files (default "./docs")
      --sitemap string     URL to sitemap.xml file
      --filter string      Filter URLs containing this path (e.g. '/docs/')
  -w, --workers int        Number of concurrent workers (default: 4, use 1 for sequential)
```

### Serve Command

```
mdify serve

Flags:
  -d, --dir string    Directory containing markdown files (default "./docs")
  -p, --port int      Port to serve on (default 8080)
```

## Examples


### Convert and Serve Locally

```bash
# Convert documentation
mdify scrape --sitemap https://example.com/sitemap.xml --selector ".content"

# Serve locally for browsing
mdify serve --port 3000
```

### Sequential Processing for Rate-Limited Sites

You can set `--workers` to 1 to process files sequentially:

```bash
mdify scrape \
  --sitemap https://example.com/sitemap.xml \
  --selector ".main-content" \
  --workers 1
```

## Output Structure

mdify preserves the URL structure as file paths. For example:

- `https://example.com/docs/getting-started` → `./docs/docs/getting-started.md`
- `https://example.com/api/reference` → `./docs/api/reference.md`
- `https://example.com/` → `./docs/index.md`

## Development

### Building

```bash
make all
```

This builds for all platforms (Windows, macOS Intel, macOS Silicon, Linux) and creates release archives.

### Testing

```bash
go test -v ./...
```

## Changelog

### 0.1.0
* Initial release
* Sitemap support with URL filtering
* Concurrent processing with configurable workers
* Built-in HTTP server for browsing converted files
* Retry logic with exponential backoff
* Comprehensive test suite
