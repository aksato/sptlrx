package mpd

import (
	"regexp"
	"github.com/raitonoberu/sptlrx/player"
	"strconv"

	"github.com/fhs/gompd/mpd"
)

// Replaces all forbidden or problematic filename characters with '-'
func sanitizeFilename(name string) string {
	// Replace '/' and null byte (shouldn't appear in Go strings, but for completeness)
	name = regexp.MustCompile(`[\/\x00]`).ReplaceAllString(name, "-")
	// Optionally, replace other problematic characters for cross-platform safety
	name = regexp.MustCompile(`[:*?"<>|\\]`).ReplaceAllString(name, "-")
	// Collapse multiple dashes
	name = regexp.MustCompile(`-+`).ReplaceAllString(name, "-")
	// Trim spaces and dashes from start/end
	name = regexp.MustCompile(`^[\s\-\.]+|[\s\-\.]+$`).ReplaceAllString(name, "")
	return name
}

func New(address, password string) *Client {
	return &Client{
		address:  address,
		password: password,
	}
}

// Client implements player.Player
type Client struct {
	address  string
	password string
	client   *mpd.Client
}

func (c *Client) connect() error {
	if c.client != nil {
		c.client.Close()
	}
	client, err := mpd.DialAuthenticated("tcp", c.address, c.password)
	if err != nil {
		c.client = nil
		return err
	}
	c.client = client
	return nil
}

func (c *Client) checkConnection() error {
	if c.client == nil || c.client.Ping() != nil {
		return c.connect()
	}
	return nil
}

func (c *Client) State() (*player.State, error) {
	if err := c.checkConnection(); err != nil {
		return nil, err
	}

	status, err := c.client.Status()
	if err != nil {
		return nil, err
	}
	current, err := c.client.CurrentSong()
	if err != nil {
		return nil, err
	}
	elapsed, _ := strconv.ParseFloat(status["elapsed"], 32)

	var title string
	if t, ok := current["Title"]; ok {
		title = t
	}

	var artist string
	if a, ok := current["Artist"]; ok {
		artist = a
	}

	var query string
	if artist != "" {
		query = artist + " - " + title
	} else {
		query = title
	}
	query = sanitizeFilename(query)

	return &player.State{
		ID:       status["songid"],
		Query:    query,
		Playing:  status["state"] == "play",
		Position: int(elapsed) * 1000,
	}, nil
}
