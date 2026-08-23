package music

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TrackInfo holds details about the currently playing track.
type TrackInfo struct {
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	State    string  `json:"state"`
	Volume   int     `json:"volume"`
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
	Shuffle  bool    `json:"shuffle"`
	Repeat   string  `json:"repeat"`
	Loved    bool    `json:"loved"`
}

// TrackStats holds extended metadata for the current track.
type TrackStats struct {
	Title     string  `json:"title"`
	Artist    string  `json:"artist"`
	Album     string  `json:"album"`
	Year      int     `json:"year"`
	Genre     string  `json:"genre"`
	Duration  float64 `json:"duration"`
	PlayCount int     `json:"play_count"`
	Rating    int     `json:"rating"` // 0–100
	Loved     bool    `json:"loved"`
	DateAdded string  `json:"date_added"`
}

// LibraryStats holds aggregate statistics for the music library.
type LibraryStats struct {
	TotalTracks     int    `json:"total_tracks"`
	TotalPlaylists  int    `json:"total_playlists"`
	MostPlayedGenre string `json:"most_played_genre"`
	TopArtist       string `json:"top_artist"`
	TotalPlayCount  int    `json:"total_play_count"`
}

// PlaylistInfo identifies a Music.app playlist without relying on its display name.
type PlaylistInfo struct {
	Name         string `json:"name"`
	ID           int    `json:"id"`
	PersistentID string `json:"persistent_id"`
	Kind         string `json:"kind"`
	TrackCount   int    `json:"track_count"`
}

const recordSep = "\x1f"

// RunAppleScript executes an AppleScript snippet and returns trimmed stdout.
func RunAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("applescript error: %w (output: %s)", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// IsMusicRunning reports whether Music.app is currently running.
// Uses AppleScript `application "Music" is running` which is the canonical
// macOS check, with a pgrep fallback for non-GUI contexts.
func IsMusicRunning() (bool, error) {
	out, err := RunAppleScript(`tell application "System Events" to get (name of processes) contains "Music"`)
	if err == nil {
		return out == "true", nil
	}
	// Fallback: pgrep (e.g. when System Events is denied)
	cmd := exec.Command("pgrep", "-x", "Music")
	err = cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// EnsureMusicRunning launches Music.app if it is not already running.
// After activation it waits briefly for the process to appear, so the
// caller can immediately issue AppleScript commands without a race.
func EnsureMusicRunning() error {
	running, err := IsMusicRunning()
	if err != nil {
		return err
	}
	if !running {
		if _, err = RunAppleScript(`tell application "Music" to activate`); err != nil {
			return err
		}
		// Give Music.app up to 5s to launch (poll every 250ms).
		for i := 0; i < 20; i++ {
			time.Sleep(250 * time.Millisecond)
			ok, _ := IsMusicRunning()
			if ok {
				break
			}
		}
	}
	return nil
}

func Play() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(`tell application "Music" to play`)
	return err
}

func Pause() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(`tell application "Music" to pause`)
	return err
}

func Toggle() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(`tell application "Music" to playpause`)
	return err
}

func Next() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	return advanceTrack("next")
}

func Prev() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	return advanceTrack("previous")
}

func advanceTrack(direction string) error {
	nativeCommand := "next track"
	step := 1
	wrapIndex := "1"
	if direction == "previous" {
		nativeCommand = "previous track"
		step = -1
		wrapIndex = "total"
	}

	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				if not (exists current track) then return "NO_TRACK"
				set curPID to persistent ID of current track
				try
					%s
					delay 0.15
					if (exists current track) and persistent ID of current track is not curPID then return "PLAYING"
				end try

				try
					set p to container of current track
					set total to count of tracks of p
					if total is 0 then return "NO_TRACK"
					repeat with i from 1 to total
						try
							if persistent ID of track i of p is curPID then
								set targetIndex to i + (%d)
								if targetIndex > total then set targetIndex to %s
								if targetIndex < 1 then set targetIndex to %s
								set targetTrack to item targetIndex of tracks of p
								set targetPID to persistent ID of targetTrack
								repeat with attempt from 1 to 4
									play targetTrack
									delay 0.25
									if (exists current track) and persistent ID of current track is targetPID then return "PLAYING"
								end repeat
								return "NO_CHANGE"
							end if
						end try
					end repeat
				end try

				try
					set p to library playlist 1
					set total to count of tracks of p
					if total is 0 then return "NO_TRACK"
					repeat with i from 1 to total
						try
							if persistent ID of track i of p is curPID then
								set targetIndex to i + (%d)
								if targetIndex > total then set targetIndex to %s
								if targetIndex < 1 then set targetIndex to %s
								set targetTrack to item targetIndex of tracks of p
								set targetPID to persistent ID of targetTrack
								repeat with attempt from 1 to 4
									play targetTrack
									delay 0.25
									if (exists current track) and persistent ID of current track is targetPID then return "PLAYING"
								end repeat
								return "NO_CHANGE"
							end if
						end try
					end repeat
				end try

				return "NO_CHANGE"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, nativeCommand, step, wrapIndex, wrapIndex, step, wrapIndex, wrapIndex))
	if err != nil {
		return err
	}
	switch {
	case out == "PLAYING":
		return nil
	case out == "NO_TRACK":
		return fmt.Errorf("no current track")
	case out == "NO_CHANGE":
		return fmt.Errorf("could not advance track")
	case strings.HasPrefix(out, "ERROR"+recordSep):
		return fmt.Errorf("applescript error: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	default:
		return nil
	}
}

func Stop() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(`tell application "Music" to stop`)
	return err
}

// GetVolume returns Music.app's own volume (0–100), not the system volume.
func GetVolume() (int, error) {
	if err := EnsureMusicRunning(); err != nil {
		return 0, err
	}
	out, err := RunAppleScript(`tell application "Music" to get sound volume`)
	if err != nil {
		return 0, err
	}
	vol, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("unexpected volume output: %q", out)
	}
	return vol, nil
}

// SetVolume sets Music.app's volume (0–100), not the system volume.
func SetVolume(vol int) error {
	if vol < 0 || vol > 100 {
		return fmt.Errorf("volume must be 0–100")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(fmt.Sprintf(`tell application "Music" to set sound volume to %d`, vol))
	return err
}

func ToggleShuffle() (bool, error) {
	if err := EnsureMusicRunning(); err != nil {
		return false, err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			set shuffle enabled to not shuffle enabled
			return shuffle enabled
		end tell
	`)
	if err != nil {
		return false, err
	}
	return out == "true", nil
}

func CycleRepeat() (string, error) {
	if err := EnsureMusicRunning(); err != nil {
		return "", err
	}
	return RunAppleScript(`
		tell application "Music"
			set cur to song repeat as string
			if cur is "off" then
				set song repeat to all
				return "all"
			else if cur is "all" or cur is "yes" then
				set song repeat to one
				return "one"
			else
				set song repeat to off
				return "off"
			end if
		end tell
	`)
}

func Seek(delta float64) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			set cur to player position
			set newPos to cur + (%f)
			if newPos < 0 then set player position to 0
			else set player position to newPos
		end tell
	`, delta))
	return err
}

// SetPlayerPosition jumps to an absolute timestamp within the current song.
func SetPlayerPosition(seconds float64) error {
	if seconds < 0 {
		seconds = 0
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			if not (exists current track) then return "NO_TRACK"
			set player position to %f
		end tell
	`, seconds))
	if err != nil {
		return err
	}
	if out == "NO_TRACK" {
		return fmt.Errorf("no current track")
	}
	return nil
}

// FadeIn gradually raises Music.app volume from 0 to targetVolume over seconds.
func FadeIn(seconds float64, targetVolume int) error {
	if seconds <= 0 || seconds != seconds || seconds > 3600 {
		seconds = 3
	}
	if targetVolume <= 0 {
		targetVolume = 80
	}
	if targetVolume > 100 {
		targetVolume = 100
	}
	steps := 20
	delay := time.Duration(float64(time.Second) * seconds / float64(steps))
	for i := 1; i <= steps; i++ {
		vol := int(float64(targetVolume) * float64(i) / float64(steps))
		if err := SetVolume(vol); err != nil {
			return err
		}
		if i < steps {
			time.Sleep(delay)
		}
	}
	return nil
}

// FadeOut gradually reduces Music.app volume to zero over the requested duration.
func FadeOut(seconds float64) error {
	if seconds <= 0 || seconds != seconds || seconds > 3600 {
		seconds = 5
	}
	startVolume, err := GetVolume()
	if err != nil {
		return err
	}
	steps := 20
	if seconds < 2 {
		steps = 10
	}
	delay := time.Duration(seconds * float64(time.Second) / float64(steps))
	for i := 1; i <= steps; i++ {
		nextVolume := startVolume - int(float64(startVolume)*float64(i)/float64(steps))
		if err := SetVolume(nextVolume); err != nil {
			return err
		}
		if i < steps {
			time.Sleep(delay)
		}
	}
	return nil
}

// SetShuffleMode enables or disables shuffle mode.
func SetShuffleMode(enabled bool) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(fmt.Sprintf(`tell application "Music" to set shuffle enabled to %t`, enabled))
	return err
}

// SetRepeatMode sets repeat mode to off, one, or all.
func SetRepeatMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "off" && mode != "one" && mode != "all" {
		return fmt.Errorf("repeat mode must be off, one, or all")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(fmt.Sprintf(`tell application "Music" to set song repeat to %s`, mode))
	return err
}

// ToggleLove toggles the loved state of the current track and returns the new state.
// Falls back gracefully for streaming tracks that may not support the loved property.
func ToggleLove() (bool, error) {
	if err := EnsureMusicRunning(); err != nil {
		return false, err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			try
				set loved of current track to not loved of current track
				return loved of current track
			on error
				return "UNSUPPORTED"
			end try
		end tell
	`)
	if err != nil {
		return false, err
	}
	if out == "UNSUPPORTED" {
		return false, fmt.Errorf("loved is not supported for this track")
	}
	return out == "true", nil
}

// SetLove sets the loved state of the current track and returns the new state.
func SetLove(loved bool) (bool, error) {
	if err := EnsureMusicRunning(); err != nil {
		return false, err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set loved of current track to %t
				return loved of current track
			on error
				return "UNSUPPORTED"
			end try
		end tell
	`, loved))
	if err != nil {
		return false, err
	}
	if out == "UNSUPPORTED" {
		return false, fmt.Errorf("loved is not supported for this track")
	}
	return out == "true", nil
}

// DislikeCurrentTrack marks the current track as disliked when Music.app exposes that property.
func DislikeCurrentTrack() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			try
				set loved of current track to false
			end try
			try
				set disliked of current track to true
				return "OK"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`)
	if err != nil {
		return err
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("disliked is not supported for this track: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// RateCurrentTrack sets the current track rating. Stars are 1-5 and map to Music.app's 20-100 scale.
func RateCurrentTrack(stars int) error {
	if stars < 1 || stars > 5 {
		return fmt.Errorf("rating must be 1-5 stars")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	_, err := RunAppleScript(fmt.Sprintf(`tell application "Music" to set rating of current track to %d`, stars*20))
	return err
}

// NowPlaying returns full playback state including loved status.
func NowPlaying() (*TrackInfo, error) {
	running, err := IsMusicRunning()
	if err != nil {
		return nil, err
	}
	if !running {
		return &TrackInfo{State: "stopped"}, nil
	}

	out, err := RunAppleScript(`
		tell application "Music"
			try
				set sep to ASCII character 31
				set tVolume to sound volume
				if player state is stopped then return "STOPPED" & sep & tVolume
				if not (exists current track) then return "STOPPED" & sep & tVolume
				set t to current track
				set tName    to name of t
				set tArtist  to artist of t
				set tAlbum   to album of t
				set tPos     to player position
				set tDur     to duration of t
				set pState   to player state as string
				set tShuffle to shuffle enabled
				set tRepeat  to song repeat as string
				set tLoved   to false
				try
					set tLoved to loved of t
				end try
				return tName & sep & tArtist & sep & tAlbum & sep & pState & sep & tVolume & sep & tPos & sep & tDur & sep & tShuffle & sep & tRepeat & sep & tLoved
			on error errText
				return "ERROR" & sep & errText
			end try
		end tell
	`)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(out, recordSep)
	if len(parts) == 0 {
		return nil, errors.New("empty response from AppleScript")
	}
	switch parts[0] {
	case "ERROR":
		return nil, fmt.Errorf("applescript: %s", parts[1])
	case "STOPPED":
		vol := 0
		if len(parts) > 1 {
			vol, _ = strconv.Atoi(parts[1])
		}
		return &TrackInfo{State: "stopped", Volume: vol}, nil
	}
	if len(parts) < 7 {
		return nil, fmt.Errorf("unexpected output: %s", out)
	}

	vol, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		vol = 0
	}
	pos, _ := strconv.ParseFloat(strings.TrimSpace(parts[5]), 64)
	dur, _ := strconv.ParseFloat(strings.TrimSpace(parts[6]), 64)

	state := strings.ToLower(parts[3])
	switch {
	case strings.Contains(state, "play"):
		state = "playing"
	case strings.Contains(state, "paus"):
		state = "paused"
	default:
		state = "stopped"
	}

	shuffle := len(parts) > 7 && parts[7] == "true"
	repeat := "off"
	if len(parts) > 8 {
		repeat = strings.ToLower(parts[8])
	}
	loved := len(parts) > 9 && parts[9] == "true"

	return &TrackInfo{
		Title: parts[0], Artist: parts[1], Album: parts[2],
		State: state, Volume: vol, Position: pos, Duration: dur,
		Shuffle: shuffle, Repeat: repeat, Loved: loved,
	}, nil
}

// GetTrackStats returns play count, rating, loved, date added, year, genre, and duration for the current track.
func GetTrackStats() (*TrackStats, error) {
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			try
				set sep to ASCII character 31
				if not (exists current track) then return "NO_TRACK"
				set t to current track
				set tCount to played count of t
				set tRating to rating of t
				set tAdded to ""
				try
					set tAdded to date added of t as string
				end try
				set tLoved to false
				try
					set tLoved to loved of t
				end try
				set tYear to 0
				try
					set tYear to year of t
				end try
				set tGenre to ""
				try
					set tGenre to genre of t
				end try
				set tDur to duration of t
				return (name of t) & sep & (artist of t) & sep & (album of t) & sep & tCount & sep & tRating & sep & tLoved & sep & tAdded & sep & tYear & sep & tGenre & sep & tDur
			on error errText
				return "ERROR" & sep & errText
			end try
		end tell
	`)
	if err != nil {
		return nil, err
	}
	if out == "NO_TRACK" {
		return nil, fmt.Errorf("no track playing")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		parts := strings.SplitN(out, recordSep, 2)
		if len(parts) > 1 {
			return nil, fmt.Errorf("applescript: %s", parts[1])
		}
		return nil, fmt.Errorf("applescript: %s", out)
	}

	parts := strings.SplitN(out, recordSep, 10)
	if len(parts) < 10 {
		return nil, fmt.Errorf("unexpected output: %s", out)
	}
	count, _ := strconv.Atoi(parts[3])
	rating, _ := strconv.Atoi(parts[4])
	year, _ := strconv.Atoi(parts[7])
	duration, _ := strconv.ParseFloat(parts[9], 64)
	return &TrackStats{
		Title:     parts[0],
		Artist:    parts[1],
		Album:     parts[2],
		Year:      year,
		Genre:     parts[8],
		Duration:  duration,
		PlayCount: count,
		Rating:    rating,
		Loved:     parts[5] == "true",
		DateAdded: parts[6],
	}, nil
}

// GetQueue returns up to 10 tracks following the current track in its playlist.
func GetQueue() ([]TrackInfo, error) {
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			try
				set sep to ASCII character 31
				set curPlaylist to container of current track
				set curPID to persistent ID of current track
				set total to count of tracks of curPlaylist
				set output to ""
				set curIdx to 0
				repeat with i from 1 to total
					try
						if persistent ID of track i of curPlaylist is curPID then
							set curIdx to i
							exit repeat
						end if
					end try
				end repeat
				if curIdx is 0 then return "ERROR" & sep & "current track not found in its playlist"
				set emitted to 0
				set i to curIdx + 1
				repeat while emitted < 10 and total > 0
					if i > total then set i to 1
					if i is curIdx then exit repeat
					set t to item i of tracks of curPlaylist
					set output to output & name of t & sep & artist of t & sep & album of t & "\n"
					set emitted to emitted + 1
					set i to i + 1
				end repeat
				return output
			on error errText
				return "ERROR" & sep & errText
			end try
		end tell
	`)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return nil, fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return parseRecordTrackLines(out, recordSep), nil
}

// GetArtworkPath saves the current track's artwork to /tmp/muse_art.jpg and returns the path.
// Returns empty string when no artwork is available.
func GetArtworkPath() (string, error) {
	if err := EnsureMusicRunning(); err != nil {
		return "", err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			try
				if not (exists current track) then return "NO_ART"
				set artList to artworks of current track
				if (count of artList) = 0 then return "NO_ART"
				set artData to raw data of item 1 of artList
				set tmpPath to (do shell script "mktemp /tmp/muse_art_XXXXXX.jpg")
				set fp to open for access POSIX file tmpPath with write permission
				try
					set eof fp to 0
					write artData to fp
					close access fp
				on error writeErr
					try
						close access fp
					end try
					do shell script "rm -f " & quoted form of tmpPath
					return "NO_ART"
				end try
				return tmpPath
			on error
				return "NO_ART"
			end try
		end tell
	`)
	if err != nil {
		return "", err
	}
	if out == "NO_ART" || out == "" {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// Search queries the library for tracks matching query in title, artist, or album.
func Search(query string) ([]TrackInfo, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	q := escapeAS(query)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			with timeout of 15 seconds
			set sep to ASCII character 31
			set output to ""
			try
				set trackList to (every track of library playlist 1 whose name contains "%s" or artist contains "%s" or album contains "%s")
				set total to count of trackList
				if total > 50 then set total to 50
				repeat with i from 1 to total
					set t to item i of trackList
					set output to output & name of t & sep & artist of t & sep & album of t & "\n"
				end repeat
			on error errText
				return "ERROR" & sep & errText
			end try
			return output
		end timeout
		end tell
	`, q, q, q))
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return nil, fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return parseRecordTrackLines(out, recordSep), nil
}

// SearchFiltered queries the library with dynamic filters. Each non-empty filter is
// combined with 'and'. When only query is provided (no specific filters), it falls
// back to the existing name/artist/album contains logic. Limit defaults to 50 (1-100).
func SearchFiltered(query, artist, album, genre string, year int, loved *bool, minRating int, limit int) ([]TrackInfo, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}

	// Build predicate list dynamically.
	var preds []string

	query = strings.TrimSpace(query)
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	genre = strings.TrimSpace(genre)

	hasSpecificFilters := artist != "" || album != "" || genre != "" || year > 0 || loved != nil || minRating > 0

	if hasSpecificFilters {
		// Use specific predicates for the provided filters.
		if artist != "" {
			preds = append(preds, fmt.Sprintf("artist contains \"%s\"", escapeAS(artist)))
		}
		if album != "" {
			preds = append(preds, fmt.Sprintf("album contains \"%s\"", escapeAS(album)))
		}
		if genre != "" {
			preds = append(preds, fmt.Sprintf("genre contains \"%s\"", escapeAS(genre)))
		}
		if year > 0 {
			preds = append(preds, fmt.Sprintf("year is %d", year))
		}
		if loved != nil {
			if *loved {
				preds = append(preds, "loved is true")
			} else {
				preds = append(preds, "loved is false")
			}
		}
		if minRating > 0 {
			// Rating stored as 0-100; stars 1-5 map to 20-100.
			preds = append(preds, fmt.Sprintf("rating >= %d", minRating*20))
		}
		// If a text query is also provided, narrow results further.
		if query != "" {
			q := escapeAS(query)
			preds = append(preds, fmt.Sprintf("(name contains \"%s\" or artist contains \"%s\" or album contains \"%s\")", q, q, q))
		}
	} else {
		// Fallback: match existing Search behaviour (name/artist/album contains query).
		if query == "" {
			return nil, fmt.Errorf("search query is required")
		}
		q := escapeAS(query)
		preds = append(preds, fmt.Sprintf("name contains \"%s\" or artist contains \"%s\" or album contains \"%s\"", q, q, q))
	}

	predicate := strings.Join(preds, " and ")

	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			with timeout of 15 seconds
			set sep to ASCII character 31
			set output to ""
			try
				set trackList to (every track of library playlist 1 whose %s)
				set total to count of trackList
				if total > %d then set total to %d
				repeat with i from 1 to total
					set t to item i of trackList
					set output to output & name of t & sep & artist of t & sep & album of t & "\n"
				end repeat
			on error errText
				return "ERROR" & sep & errText
			end try
			return output
		end timeout
		end tell
	`, predicate, limit, limit))
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return nil, fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return parseRecordTrackLines(out, recordSep), nil
}

// PlayTrackByName plays the first track whose title, artist, or album contains name.
func PlayTrackByName(name string) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	n := escapeAS(name)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set targetTrack to missing value
				set t1 to (every track of library playlist 1 whose name contains "%s")
				if (count of t1) > 0 then
					set targetTrack to item 1 of t1
				else
					set t2 to (every track of library playlist 1 whose artist contains "%s")
					if (count of t2) > 0 then
						set targetTrack to item 1 of t2
					else
						set t3 to (every track of library playlist 1 whose album contains "%s")
						if (count of t3) > 0 then
							set targetTrack to item 1 of t3
						end if
					end if
				end if
				if targetTrack is missing value then return "NOT_FOUND"
				set targetPID to persistent ID of targetTrack
				repeat with attempt from 1 to 4
					play targetTrack
					delay 0.25
					if (exists current track) and persistent ID of current track is targetPID then return "PLAYING"
				end repeat
				return "NO_CHANGE"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, n, n, n))
	if err != nil {
		return err
	}
	if out == "NOT_FOUND" {
		return fmt.Errorf("no track found matching: %s", name)
	}
	if out == "NO_CHANGE" {
		return fmt.Errorf("could not play track: %s", name)
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// PlayPlaylist plays the named playlist from the beginning.
func PlayPlaylist(name string) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	n := escapeAS(name)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				if exists playlist "%s" then
					play playlist "%s"
					return "PLAYING"
				else
					return "NOT_FOUND"
				end if
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, n, n))
	if err != nil {
		return err
	}
	if out == "NOT_FOUND" {
		return fmt.Errorf("no playlist found: %s", name)
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// PlayPlaylistByPersistentID plays a specific playlist from the beginning.
func PlayPlaylistByPersistentID(persistentID string) error {
	return playPlaylistByPersistentID(persistentID, false)
}

// GetPlaylists returns the names of all playlists (used by MCP).
func GetPlaylists() ([]string, error) {
	mine, others, err := GetCategorizedPlaylistInfos()
	if err != nil {
		return nil, err
	}
	all := append(mine, others...)
	names := make([]string, 0, len(all))
	for _, p := range all {
		names = append(names, p.Name)
	}
	return names, nil
}

// GetCategorizedPlaylists returns playlists split into user-created and subscription/shared.
func GetCategorizedPlaylists() (mine []string, others []string, err error) {
	mineInfo, othersInfo, err := GetCategorizedPlaylistInfos()
	if err != nil {
		return nil, nil, err
	}
	for _, p := range mineInfo {
		mine = append(mine, p.Name)
	}
	for _, p := range othersInfo {
		others = append(others, p.Name)
	}
	return mine, others, nil
}

// GetCategorizedPlaylistInfos returns playable playlists split into user/library
// playlists and Apple Music/shared playlists. Folder playlists without direct
// tracks are skipped because opening them as track lists produces empty screens.
func GetCategorizedPlaylistInfos() (mine []PlaylistInfo, others []PlaylistInfo, err error) {
	if err = EnsureMusicRunning(); err != nil {
		return
	}
	out, err := RunAppleScript(`
		tell application "Music"
			set sep to ASCII character 31
			set nl to ASCII character 10
			set output to ""
			try
				repeat with p in every playlist
					try
						set pClass to class of p as string
						set pName to name of p
						set pID to id of p
						set pPersistentID to persistent ID of p
						set pTrackCount to 0
						try
							set pTrackCount to count of tracks of p
						end try

						if pClass is "folder playlist" and pTrackCount is 0 then
							-- skip empty folders; they are containers, not playable track lists
						else if pClass is "user playlist" or pClass is "library playlist" or pClass is "folder playlist" then
							set output to output & "MINE" & sep & pID & sep & pPersistentID & sep & pClass & sep & pTrackCount & sep & pName & nl
						else
							set output to output & "OTHER" & sep & pID & sep & pPersistentID & sep & pClass & sep & pTrackCount & sep & pName & nl
						end if
					end try
				end repeat
			on error errText
				return "ERROR" & sep & errText
			end try
			return output
		end tell
	`)
	if err != nil {
		return
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return nil, nil, fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, recordSep, 6)
		if len(parts) < 6 {
			continue
		}
		id, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		trackCount, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
		info := PlaylistInfo{
			ID:           id,
			PersistentID: strings.TrimSpace(parts[2]),
			Kind:         strings.TrimSpace(parts[3]),
			TrackCount:   trackCount,
			Name:         parts[5],
		}
		if parts[0] == "MINE" {
			mine = append(mine, info)
		} else if parts[0] == "OTHER" {
			others = append(others, info)
		}
	}
	return
}

// GetPlaylistTracks returns up to 100 tracks from the named playlist.
func GetPlaylistTracks(playlistName string) ([]TrackInfo, error) {
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	n := escapeAS(playlistName)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose name is "%s")
				if (count of matches) is 0 then return "NOT_FOUND"
				set p to item 1 of matches
				%s
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, n, playlistTracksAppleScriptBody()))
	if err != nil {
		return nil, err
	}
	return parsePlaylistTrackOutput(out)
}

// CreatePlaylist creates a new user playlist.
func CreatePlaylist(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("playlist name is required")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				if exists playlist "%s" then return "EXISTS"
				make new user playlist with properties {name:"%s"}
				return "OK"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, escapeAS(name), escapeAS(name)))
	if err != nil {
		return err
	}
	if out == "EXISTS" {
		return fmt.Errorf("playlist already exists: %s", name)
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// AddTrackToPlaylist duplicates the first matching library track into a playlist.
func AddTrackToPlaylist(playlistName, query string) error {
	playlistName = strings.TrimSpace(playlistName)
	query = strings.TrimSpace(query)
	if playlistName == "" || query == "" {
		return fmt.Errorf("playlist name and query are required")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	q := escapeAS(query)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				if not (exists playlist "%s") then return "PLAYLIST_NOT_FOUND"
				set targetTrack to missing value
				set matches to (every track of library playlist 1 whose name contains "%s" or artist contains "%s" or album contains "%s")
				if (count of matches) > 0 then set targetTrack to item 1 of matches
				if targetTrack is missing value then return "TRACK_NOT_FOUND"
				duplicate targetTrack to playlist "%s"
				return "OK"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, escapeAS(playlistName), q, q, q, escapeAS(playlistName)))
	if err != nil {
		return err
	}
	switch out {
	case "PLAYLIST_NOT_FOUND":
		return fmt.Errorf("playlist not found: %s", playlistName)
	case "TRACK_NOT_FOUND":
		return fmt.Errorf("no library track found matching: %s", query)
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// RemoveTrackFromPlaylist removes a matching track from a playlist. If index is positive,
// it removes that 1-based position; otherwise it removes the first track matching query.
func RemoveTrackFromPlaylist(playlistName, query string, index int) error {
	playlistName = strings.TrimSpace(playlistName)
	query = strings.TrimSpace(query)
	if playlistName == "" {
		return fmt.Errorf("playlist name is required")
	}
	if index <= 0 && query == "" {
		return fmt.Errorf("query or 1-based index is required")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	var selector string
	if index > 0 {
		selector = fmt.Sprintf(`
				if %d > (count of tracks of p) then return "TRACK_NOT_FOUND"
				delete track %d of p
		`, index, index)
	} else {
		q := escapeAS(query)
		selector = fmt.Sprintf(`
				set matches to (every track of p whose name contains "%s" or artist contains "%s" or album contains "%s")
				if (count of matches) is 0 then return "TRACK_NOT_FOUND"
				delete item 1 of matches
		`, q, q, q)
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose name is "%s")
				if (count of matches) is 0 then return "PLAYLIST_NOT_FOUND"
				set p to item 1 of matches
				%s
				return "OK"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, escapeAS(playlistName), selector))
	if err != nil {
		return err
	}
	switch out {
	case "PLAYLIST_NOT_FOUND":
		return fmt.Errorf("playlist not found: %s", playlistName)
	case "TRACK_NOT_FOUND":
		return fmt.Errorf("playlist track not found")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// GetPlaylistTracksByPersistentID returns up to 100 tracks from a specific playlist.
func GetPlaylistTracksByPersistentID(persistentID string) ([]TrackInfo, error) {
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	pid := escapeAS(persistentID)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose persistent ID is "%s")
				if (count of matches) is 0 then return "NOT_FOUND"
				set p to item 1 of matches
				%s
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, pid, playlistTracksAppleScriptBody()))
	if err != nil {
		return nil, err
	}
	return parsePlaylistTrackOutput(out)
}

func playlistTracksAppleScriptBody() string {
	return `
				set sep to ASCII character 31
				set nl to ASCII character 10
				set total to count of tracks of p
				if total > 100 then set total to 100
				set output to ""
				repeat with i from 1 to total
					set t to track i of p
					set tName to ""
					set tArtist to ""
					set tAlbum to ""
					try
						set tName to name of t
					end try
					try
						set tArtist to artist of t
					end try
					try
						set tAlbum to album of t
					end try
					if tName is not "" then
						set output to output & tName & sep & tArtist & sep & tAlbum & nl
					end if
				end repeat
				return output
	`
}

func parsePlaylistTrackOutput(out string) ([]TrackInfo, error) {
	switch {
	case out == "NOT_FOUND":
		return nil, fmt.Errorf("playlist not found")
	case strings.HasPrefix(out, "ERROR"+recordSep):
		return nil, fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	default:
		return parseRecordTrackLines(out, recordSep), nil
	}
}

// PlayPlaylistShuffled enables shuffle and plays the named playlist.
func PlayPlaylistShuffled(playlistName string) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	n := escapeAS(playlistName)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set shuffle enabled to true
				play playlist "%s"
				return "PLAYING"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, n))
	if err != nil {
		return err
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// PlayPlaylistShuffledByPersistentID enables shuffle and plays a specific playlist.
func PlayPlaylistShuffledByPersistentID(persistentID string) error {
	return playPlaylistByPersistentID(persistentID, true)
}

func playPlaylistByPersistentID(persistentID string, shuffled bool) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	pid := escapeAS(persistentID)
	shuffleLine := ""
	if shuffled {
		shuffleLine = "set shuffle enabled to true"
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose persistent ID is "%s")
				if (count of matches) is 0 then return "NOT_FOUND"
				%s
				play item 1 of matches
				return "PLAYING"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, pid, shuffleLine))
	if err != nil {
		return err
	}
	if out == "NOT_FOUND" {
		return fmt.Errorf("playlist not found")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// PlayPlaylistTrack plays a specific track by title within a playlist.
func PlayPlaylistTrack(playlistName, trackTitle string) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set targetTrack to (first track of playlist "%s" whose name is "%s")
				set targetPID to persistent ID of targetTrack
				repeat with attempt from 1 to 4
					play targetTrack
					delay 0.25
					if (exists current track) and persistent ID of current track is targetPID then return "PLAYING"
				end repeat
				return "NO_CHANGE"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, escapeAS(playlistName), escapeAS(trackTitle)))
	if err != nil {
		return err
	}
	if out == "NO_CHANGE" {
		return fmt.Errorf("could not play playlist track")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// PlayPlaylistTrackByPersistentID plays a specific track by title within a playlist.
func PlayPlaylistTrackByPersistentID(persistentID, trackTitle string) error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose persistent ID is "%s")
				if (count of matches) is 0 then return "NOT_FOUND"
				set p to item 1 of matches
				set targetTrack to (first track of p whose name is "%s")
				set targetPID to persistent ID of targetTrack
				repeat with attempt from 1 to 4
					play targetTrack
					delay 0.25
					if (exists current track) and persistent ID of current track is targetPID then return "PLAYING"
				end repeat
				return "NO_CHANGE"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, escapeAS(persistentID), escapeAS(trackTitle)))
	if err != nil {
		return err
	}
	if out == "NOT_FOUND" {
		return fmt.Errorf("playlist not found")
	}
	if out == "NO_CHANGE" {
		return fmt.Errorf("could not play playlist track")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// PlayPlaylistTrackAtIndexByPersistentID plays a specific 0-based track index
// within a playlist. This is preferred by the TUI because duplicate track names
// are common in real libraries.
func PlayPlaylistTrackAtIndexByPersistentID(persistentID string, index int) error {
	if index < 0 {
		return fmt.Errorf("track index must be non-negative")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose persistent ID is "%s")
				if (count of matches) is 0 then return "NOT_FOUND"
				set p to item 1 of matches
				set oneBasedIndex to %d
				if oneBasedIndex > (count of tracks of p) then return "NOT_FOUND"
				set targetTrack to item oneBasedIndex of tracks of p
				set targetPID to persistent ID of targetTrack
				repeat with attempt from 1 to 4
					play targetTrack
					delay 0.25
					if (exists current track) and persistent ID of current track is targetPID then return "PLAYING"
				end repeat
				return "NO_CHANGE"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, escapeAS(persistentID), index+1))
	if err != nil {
		return err
	}
	if out == "NOT_FOUND" {
		return fmt.Errorf("playlist track not found")
	}
	if out == "NO_CHANGE" {
		return fmt.Errorf("could not play playlist track")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// PlayPlaylistTrackAtIndex plays a specific 1-based track index within a named playlist.
func PlayPlaylistTrackAtIndex(playlistName string, index int) error {
	if strings.TrimSpace(playlistName) == "" {
		return fmt.Errorf("playlist name is required")
	}
	if index <= 0 {
		return fmt.Errorf("track index must be 1 or greater")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose name is "%s")
				if (count of matches) is 0 then return "PLAYLIST_NOT_FOUND"
				set p to item 1 of matches
				if %d > (count of tracks of p) then return "TRACK_NOT_FOUND"
				set targetTrack to item %d of tracks of p
				set targetPID to persistent ID of targetTrack
				repeat with attempt from 1 to 4
					play targetTrack
					delay 0.25
					if (exists current track) and persistent ID of current track is targetPID then return "PLAYING"
				end repeat
				return "NO_CHANGE"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, escapeAS(playlistName), index, index))
	if err != nil {
		return err
	}
	switch out {
	case "PLAYLIST_NOT_FOUND":
		return fmt.Errorf("playlist not found: %s", playlistName)
	case "TRACK_NOT_FOUND":
		return fmt.Errorf("playlist track not found")
	case "NO_CHANGE":
		return fmt.Errorf("could not play playlist track")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// AddCurrentTrackToLibrary duplicates the current track into the library playlist when supported.
func AddCurrentTrackToLibrary() error {
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			try
				if not (exists current track) then return "NO_TRACK"
				duplicate current track to library playlist 1
				return "OK"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`)
	if err != nil {
		return err
	}
	if out == "NO_TRACK" {
		return fmt.Errorf("no current track")
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// GetRecentlyPlayed returns tracks from Music.app's Recently Played playlist when present.
func GetRecentlyPlayed(limit int) ([]TrackInfo, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every playlist whose name is "Recently Played")
				if (count of matches) is 0 then return "NOT_FOUND"
				set p to item 1 of matches
				set sep to ASCII character 31
				set nl to ASCII character 10
				set output to ""
				set total to count of tracks of p
				if total > %d then set total to %d
				repeat with i from 1 to total
					set t to track i of p
					set output to output & name of t & sep & artist of t & sep & album of t & nl
				end repeat
				return output
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, limit, limit))
	if err != nil {
		return nil, err
	}
	return parsePlaylistTrackOutput(out)
}

// GetTopTracks returns the most-played library tracks.
func GetTopTracks(limit int) ([]TrackStats, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			with timeout of 30 seconds
				set sep to ASCII character 31
				set nl to ASCII character 10
				set output to ""
				try
					-- Cap iteration to avoid AppleScript string/time limits on huge libraries.
					set allTracks to every track of library playlist 1
					set total to count of allTracks
					if total > 2000 then set total to 2000
					repeat with i from 1 to total
						try
							set t to item i of allTracks
							set tYear to 0
							try
								set tYear to year of t
							end try
							set tGenre to ""
							try
								set tGenre to genre of t
							end try
							set output to output & (name of t) & sep & (artist of t) & sep & (album of t) & sep & (played count of t) & sep & (rating of t) & sep & tYear & sep & tGenre & sep & (duration of t) & nl
						end try
					end repeat
				on error errText
					return "ERROR" & sep & errText
				end try
				return output
			end timeout
		end tell
	`)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return nil, fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	var tracks []TrackStats
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, recordSep, 8)
		if len(parts) < 8 {
			continue
		}
		playCount, _ := strconv.Atoi(parts[3])
		rating, _ := strconv.Atoi(parts[4])
		year, _ := strconv.Atoi(parts[5])
		duration, _ := strconv.ParseFloat(parts[7], 64)
		tracks = append(tracks, TrackStats{
			Title:     parts[0],
			Artist:    parts[1],
			Album:     parts[2],
			Year:      year,
			Genre:     parts[6],
			Duration:  duration,
			PlayCount: playCount,
			Rating:    rating,
		})
	}
	sort.SliceStable(tracks, func(i, j int) bool {
		return tracks[i].PlayCount > tracks[j].PlayCount
	})
	if len(tracks) > limit {
		tracks = tracks[:limit]
	}
	return tracks, nil
}

// GetLibraryStats returns aggregate statistics for the entire music library.
func GetLibraryStats() (*LibraryStats, error) {
	if err := EnsureMusicRunning(); err != nil {
		return nil, err
	}
	out, err := RunAppleScript(`
		tell application "Music"
			with timeout of 60 seconds
				set sep to ASCII character 31
				try
					set lib to library playlist 1
					set totalTracks to count of tracks of lib
					set totalPlaylists to count of playlists
					set genreCounts to {}
					set artistCounts to {}
					set totalPlayCount to 0
					if totalTracks > 5000 then set totalTracks to 5000
					set allTracks to every track of lib
					set total to count of allTracks
					if total > 5000 then set total to 5000
					repeat with i from 1 to total
						try
							set t to item i of allTracks
							set totalPlayCount to totalPlayCount + (played count of t)
							try
								set g to genre of t
								if g is not "" then
									set found to false
									repeat with j from 1 to count of genreCounts
										if item 1 of item j of genreCounts is g then
											set item 2 of item j of genreCounts to (item 2 of item j of genreCounts) + 1
											set found to true
											exit repeat
										end if
									end repeat
									if not found then set end of genreCounts to {g, 1}
								end if
							end try
							try
								set a to artist of t
								if a is not "" then
									set found to false
									repeat with j from 1 to count of artistCounts
										if item 1 of item j of artistCounts is a then
											set item 2 of item j of artistCounts to (item 2 of item j of artistCounts) + 1
											set found to true
											exit repeat
										end if
									end repeat
									if not found then set end of artistCounts to {a, 1}
								end if
							end try
						end try
					end repeat
					set topGenre to ""
					set topGenreCount to 0
					repeat with j from 1 to count of genreCounts
						if item 2 of item j of genreCounts > topGenreCount then
							set topGenre to item 1 of item j of genreCounts
							set topGenreCount to item 2 of item j of genreCounts
						end if
					end repeat
					set topArtist to ""
					set topArtistCount to 0
					repeat with j from 1 to count of artistCounts
						if item 2 of item j of artistCounts > topArtistCount then
							set topArtist to item 1 of item j of artistCounts
							set topArtistCount to item 2 of item j of artistCounts
						end if
					end repeat
					return totalTracks & sep & totalPlaylists & sep & topGenre & sep & topArtist & sep & totalPlayCount
				on error errText
					return "ERROR" & sep & errText
				end try
			end timeout
		end tell
	`)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return nil, fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	parts := strings.SplitN(out, recordSep, 5)
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected output: %s", out)
	}
	totalTracks, _ := strconv.Atoi(parts[0])
	totalPlaylists, _ := strconv.Atoi(parts[1])
	totalPlayCount, _ := strconv.Atoi(parts[4])
	return &LibraryStats{
		TotalTracks:     totalTracks,
		TotalPlaylists:  totalPlaylists,
		MostPlayedGenre: parts[2],
		TopArtist:       parts[3],
		TotalPlayCount:  totalPlayCount,
	}, nil
}
func PlayAlbumByName(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("album query is required")
	}
	if err := EnsureMusicRunning(); err != nil {
		return err
	}
	q := escapeAS(query)
	out, err := RunAppleScript(fmt.Sprintf(`
		tell application "Music"
			try
				set matches to (every track of library playlist 1 whose album contains "%s")
				if (count of matches) is 0 then return "NOT_FOUND"
				set firstTrack to item 1 of matches
				set albumName to album of firstTrack
				set albumArtist to artist of firstTrack
				set albumTracks to (every track of library playlist 1 whose album is albumName and artist is albumArtist)
				if (count of albumTracks) is 0 then return "NOT_FOUND"
				play item 1 of albumTracks
				return "PLAYING"
			on error errText
				return "ERROR" & (ASCII character 31) & errText
			end try
		end tell
	`, q))
	if err != nil {
		return err
	}
	if out == "NOT_FOUND" {
		return fmt.Errorf("no album found matching: %s", query)
	}
	if strings.HasPrefix(out, "ERROR"+recordSep) {
		return fmt.Errorf("applescript: %s", strings.TrimPrefix(out, "ERROR"+recordSep))
	}
	return nil
}

// ExportPlaylist exports a playlist's tracks as M3U or JSON content.
// format must be "m3u" or "json".
func ExportPlaylist(name, format string) (string, error) {
	tracks, err := GetPlaylistTracks(name)
	if err != nil {
		return "", err
	}
	if len(tracks) == 0 {
		return "", fmt.Errorf("playlist is empty or not found: %s", name)
	}

	switch strings.ToLower(format) {
	case "m3u", "":
		var sb strings.Builder
		sb.WriteString("#EXTM3U\n")
		for _, t := range tracks {
			durSec := int(t.Duration)
			if durSec < 0 {
				durSec = 0
			}
			sb.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", durSec, t.Artist, t.Title))
			// Use a descriptive placeholder path; M3U consumers can resolve via metadata.
			sb.WriteString(fmt.Sprintf("%s - %s.m4a\n", t.Artist, t.Title))
		}
		return sb.String(), nil

	case "json":
		bz, err := json.MarshalIndent(tracks, "", "  ")
		if err != nil {
			return "", fmt.Errorf("json marshal: %w", err)
		}
		return string(bz), nil

	default:
		return "", fmt.Errorf("unsupported format %q (use \"m3u\" or \"json\")", format)
	}
}

// ImportPlaylist reads a playlist file (m3u or json) and adds every track it
// references to the named playlist in Music.app. The playlist is created if it
// does not already exist. filePath can also be raw content when the caller has
// already loaded the file.
func ImportPlaylist(name, filePath string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("playlist name is required")
	}

	// Determine whether filePath is a file path or raw content.
	// Heuristic: if it looks like a file path that exists on disk, read it;
	// otherwise treat the string as the file content itself.
	var content string
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		bz, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		content = string(bz)
	} else {
		// Treat as raw content
		content = filePath
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		// Try to detect format from content
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "#EXTM3U") {
			ext = ".m3u"
		} else if strings.HasPrefix(trimmed, "[") {
			ext = ".json"
		} else {
			return fmt.Errorf("cannot determine format; use a .m3u/.json file or pass raw content with detectable format")
		}
	}

	var queries []string

	switch ext {
	case ".m3u", ".m3u8":
		// Parse M3U: extract "Artist - Title" from #EXTINF lines
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#EXTINF:") {
				// Format: #EXTINF:duration,Artist - Title
				parts := strings.SplitN(line, ":", 2)
				if len(parts) < 2 {
					continue
				}
				meta := parts[1]
				if idx := strings.Index(meta, ","); idx >= 0 {
					meta = meta[idx+1:]
				}
				meta = strings.TrimSpace(meta)
				if meta != "" {
					queries = append(queries, meta)
				}
			}
		}

	case ".json":
		var tracks []TrackInfo
		if err := json.Unmarshal([]byte(content), &tracks); err != nil {
			return fmt.Errorf("parsing JSON playlist: %w", err)
		}
		for _, t := range tracks {
			q := strings.TrimSpace(t.Title)
			if q == "" {
				continue
			}
			if t.Artist != "" {
				q = t.Artist + " " + q
			}
			queries = append(queries, q)
		}

	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	if len(queries) == 0 {
		return fmt.Errorf("no tracks found in the playlist file")
	}

	// Ensure playlist exists (create if not found).
	createErr := CreatePlaylist(name)
	// CreatePlaylist returns an error if it already exists — that's fine.
	_ = createErr

	// Add each track. We tolerate individual track-not-found errors.
	var added int
	for _, q := range queries {
		if err := AddTrackToPlaylist(name, q); err != nil {
			// Skip tracks that can't be found in the library
			continue
		}
		added++
	}

	if added == 0 {
		return fmt.Errorf("could not add any tracks from the import file to playlist %q", name)
	}
	return nil
}

// escapeAS escapes backslashes and double-quotes for embedding in AppleScript strings.
func escapeAS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// parseTrackLines splits pipe-delimited newline-separated track output.
func parseTrackLines(out string) []TrackInfo {
	return parseRecordTrackLines(out, "|")
}

func parseRecordTrackLines(out, sep string) []TrackInfo {
	var tracks []TrackInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, sep, 3)
		if len(parts) >= 3 {
			tracks = append(tracks, TrackInfo{Title: parts[0], Artist: parts[1], Album: parts[2]})
		}
	}
	return tracks
}
