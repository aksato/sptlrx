package local

import (
	"bufio"
	"fmt"
	"github.com/raitonoberu/sptlrx/lyrics"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var replacer = strings.NewReplacer(
	"_", " ", "-", " ",
	",", "", ".", "",
	"!", "", "?", "",
	"(", "", ")", "",
	"[", "", "]", "",
)

type file struct {
	Path      string
	NameParts []string
}

func New(folder string) (*Client, error) {
	index, err := createIndex(folder)
	if err != nil {
		return nil, err
	}

	// Build a default logger under ~/.cache
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "sptlrx")
	_ = os.MkdirAll(cacheDir, 0o755)
	logFile := filepath.Join(cacheDir, "local_provider.log")
	f, _ := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)

	return &Client{index: index, logger: f}, nil
}

// Client implements lyrics.Provider
type Client struct {
	index  []*file
	logger io.Writer
}

func (c *Client) Lyrics(id, query string) ([]lyrics.Line, error) {
	// Simple exact match using filename
	for _, f := range c.index {
		filename := strings.TrimSuffix(filepath.Base(f.Path), ".lrc")
		if filename == query {
			fmt.Fprintf(c.logger, "Exact match found: %s\n", f.Path)

			reader, err := os.Open(f.Path)
			if err != nil {
				return nil, err
			}
			defer reader.Close()

			return parseLrcFile(reader), nil
		}
	}

	fmt.Fprintf(c.logger, "No match found for: %q\n", query)
	return nil, nil
}

func createIndex(folder string) ([]*file, error) {
	if strings.HasPrefix(folder, "~/") {
		dirname, _ := os.UserHomeDir()
		folder = filepath.Join(dirname, folder[2:])
	}

	index := []*file{}
	return index, filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if d == nil {
			return fmt.Errorf("invalid path: %s", path)
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".lrc") {
			return nil
		}

		index = append(index, &file{
			Path: path,
			// No need for NameParts since we're doing exact matching
		})
		return nil
	})
}

func parseLrcFile(reader io.Reader) []lyrics.Line {
	result := []lyrics.Line{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "[") && len(line) >= 10 {
			// Check if the second character is a digit (indicating a time tag)
			if len(line) > 1 && line[1] >= '0' && line[1] <= '9' {
				result = append(result, parseLrcLine(line))
			}
    }
	}
	return result
}

func parseLrcLine(line string) lyrics.Line {
	// [00:00.00]text -> {"time": 0, "words": "text"}
	h, _ := strconv.Atoi(line[1:3])
	m, _ := strconv.Atoi(line[4:6])
	s, _ := strconv.Atoi(line[7:9])

	return lyrics.Line{
		Time:  h*60*1000 + m*1000 + s*10,
		Words: line[10:],
	}
}
