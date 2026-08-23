package lyrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Line is a single lyric line with optional timestamp in seconds.
// Time == 0 for plain (unsynced) lyrics.
type Line struct {
	Time float64
	Text string
}

type lrclibResp struct {
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

// Fetch retrieves lyrics from lrclib.net.
// Prefers synced (timestamped) lyrics; falls back to plain.
// Returns nil, nil when the track simply has no lyrics entry.
func Fetch(title, artist, album string, duration float64) ([]Line, error) {
	params := url.Values{}
	params.Set("track_name", title)
	params.Set("artist_name", artist)
	if album != "" {
		params.Set("album_name", album)
	}
	if duration > 0 {
		params.Set("duration", strconv.Itoa(int(duration)))
	}

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get("https://lrclib.net/api/get?" + params.Encode())
	if err != nil {
		if strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "timeout") {
			return nil, fmt.Errorf("lyrics service timed out — check your internet connection")
		}
		return nil, fmt.Errorf("could not reach lyrics service")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var data lrclibResp
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if data.SyncedLyrics != "" {
		return parseLRC(data.SyncedLyrics), nil
	}
	if data.PlainLyrics != "" {
		return parsePlain(data.PlainLyrics), nil
	}
	return nil, nil
}

// lrcRe matches a single timestamp like [02:14.53] or [02:14.531]
var lrcRe = regexp.MustCompile(`\[(\d+):(\d+)\.(\d+)\]`)

func parseLRC(raw string) []Line {
	var lines []Line
	for _, l := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		matches := lrcRe.FindAllStringSubmatch(trimmed, -1)
		if len(matches) == 0 {
			continue
		}
		// Text is the content after the last ] — strip all leading timestamps.
		text := lrcRe.ReplaceAllString(trimmed, "")
		text = strings.TrimSpace(text)
		for _, m := range matches {
			min, _ := strconv.ParseFloat(m[1], 64)
			sec, _ := strconv.ParseFloat(m[2], 64)
			fracStr := m[3]
			frac, _ := strconv.ParseFloat(fracStr, 64)
			// Fractional part may be centiseconds (2 digits) or milliseconds (3 digits).
			// Divide by 10^len(fracStr) so both [00:10.50] (50/100=0.5) and
			// [00:10.123] (123/1000=0.123) are correct.
			div := 100.0
			if len(fracStr) == 3 {
				div = 1000.0
			} else if len(fracStr) != 2 {
				// Generic fallback for unexpected lengths (1 or >3 digits).
				div = 1
				for i := 0; i < len(fracStr); i++ {
					div *= 10
				}
			}
			t := min*60 + sec + frac/div
			lines = append(lines, Line{Time: t, Text: text})
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].Time < lines[j].Time })
	return lines
}

func parsePlain(raw string) []Line {
	var lines []Line
	for _, l := range strings.Split(raw, "\n") {
		lines = append(lines, Line{Text: strings.TrimSpace(l)})
	}
	return lines
}

// IsSynced reports whether the lines have timestamps (LRC format).
// A file is considered synced if any line has a non-zero timestamp —
// checking only the last line fails when the final entry is an empty
// trailing line with Time==0.
func IsSynced(lines []Line) bool {
	for _, l := range lines {
		if l.Time > 0 {
			return true
		}
	}
	return false
}

// ActiveIndex returns the index of the currently playing lyric line.
func ActiveIndex(lines []Line, pos float64) int {
	if len(lines) == 0 || !IsSynced(lines) {
		return 0
	}
	idx := 0
	for i, l := range lines {
		if l.Time <= pos {
			idx = i
		}
	}
	return idx
}
