package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/config"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/mcp"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/music"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/tui"
	"github.com/spf13/cobra"
)

var jsonOutput bool
var searchJsonOutput bool
var searchArtist string
var searchAlbum string
var searchGenre string
var searchYear int
var searchLoved bool
var searchLovedSet bool
var searchMinRating int
var searchLimit int
var statsJsonOutput bool
var queueJsonOutput bool
var playlistsJsonOutput bool
var playlistShuffle bool

// Version is overridden by release builds with -ldflags "-X github.com/ryanrodrigues25200525-svg/Apple-music-cli/cmd.Version=vX.Y.Z".
var Version = "dev"

func init() {
	nowCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	searchCmd.Flags().BoolVar(&searchJsonOutput, "json", false, "Output in JSON format")
	searchCmd.Flags().StringVar(&searchArtist, "artist", "", "Filter by artist name")
	searchCmd.Flags().StringVar(&searchAlbum, "album", "", "Filter by album name")
	searchCmd.Flags().StringVar(&searchGenre, "genre", "", "Filter by genre")
	searchCmd.Flags().IntVar(&searchYear, "year", 0, "Filter by release year")
	searchCmd.Flags().BoolVar(&searchLoved, "loved", false, "Filter by loved status")
	searchCmd.Flags().BoolVar(&searchLovedSet, "loved-set", false, "Indicates --loved was explicitly provided")
	searchCmd.Flags().IntVar(&searchMinRating, "min-rating", 0, "Filter by minimum rating (1-5 stars)")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "Maximum results (1-100)")
	statsCmd.Flags().BoolVar(&statsJsonOutput, "json", false, "Output in JSON format")
	queueCmd.Flags().BoolVar(&queueJsonOutput, "json", false, "Output in JSON format")
	queueCmd.AddCommand(queueAddCmd)
	queueCmd.AddCommand(queueClearCmd)
	queueCmd.AddCommand(queueMoveCmd)
	playlistsCmd.Flags().BoolVar(&playlistsJsonOutput, "json", false, "Output in JSON format")
	playlistPlayCmd.Flags().BoolVar(&playlistShuffle, "shuffle", false, "Enable shuffle before playing the playlist")
	playlistExportCmd.Flags().StringVar(&playlistExportFormat, "format", "m3u", "Export format: m3u or json")
	playlistExportCmd.Flags().StringVar(&playlistExportOutput, "output", "", "Write output to a file instead of stdout")

	playlistCmd.AddCommand(playlistPlayCmd)
	playlistCmd.AddCommand(playlistExportCmd)
	playlistCmd.AddCommand(playlistImportCmd)

	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(prevCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(nowCmd)
	rootCmd.AddCommand(volumeCmd)
	rootCmd.AddCommand(seekCmd)
	rootCmd.AddCommand(shuffleCmd)
	rootCmd.AddCommand(repeatCmd)
	rootCmd.AddCommand(playlistCmd)
	rootCmd.AddCommand(playlistsCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(loveCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(shareCmd)
	rootCmd.AddCommand(sleepCmd)
	rootCmd.AddCommand(queueCmd)
	rootCmd.AddCommand(miniCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

// ── Playback ──────────────────────────────────────────────────────────────────

var playCmd = &cobra.Command{
	Use:   "play [query]",
	Short: "Start playback, or search and play a track by name",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			query := strings.Join(args, " ")
			fmt.Printf("Searching for: %s...\n", query)
			if err := music.PlayTrackByName(query); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := music.Play(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("▶ Playing")
		}
	},
}

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause playback",
	Run: func(cmd *cobra.Command, args []string) {
		if err := music.Pause(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("⏸ Paused")
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle play / pause",
	Run: func(cmd *cobra.Command, args []string) {
		if err := music.Toggle(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("⏯ Toggled")
	},
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Skip to next track",
	Run: func(cmd *cobra.Command, args []string) {
		if err := music.Next(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("⏭ Next track")
	},
}

var prevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Skip to previous track",
	Run: func(cmd *cobra.Command, args []string) {
		if err := music.Prev(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("⏮ Previous track")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop playback",
	Run: func(cmd *cobra.Command, args []string) {
		if err := music.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("⏹ Stopped")
	},
}

// ── Now Playing ───────────────────────────────────────────────────────────────

var nowCmd = &cobra.Command{
	Use:   "now",
	Short: "Show the currently playing track",
	Run: func(cmd *cobra.Command, args []string) {
		info, err := music.NowPlaying()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			bz, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(bz))
			return
		}
		if info.State == "stopped" {
			fmt.Println("⏹ Stopped")
			return
		}
		stateEmoji := "▶"
		if info.State == "paused" {
			stateEmoji = "⏸"
		}
		state := info.State
		if len(state) > 0 {
			state = strings.ToUpper(state[:1]) + state[1:]
		}
		loved := ""
		if info.Loved {
			loved = "  ♥"
		}
		fmt.Printf("🎵 %s%s\n👤 %s\n💿 %s\n%s %s\n♪  %d%%\n",
			info.Title, loved, info.Artist, info.Album, stateEmoji, state, info.Volume)
	},
}

// ── Volume ────────────────────────────────────────────────────────────────────

var volumeCmd = &cobra.Command{
	Use:   "volume [0-100]",
	Short: "Get or set Music.app volume",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			vol, err := music.GetVolume()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("♪  Volume: %d%%\n", vol)
			return
		}
		vol, err := strconv.Atoi(args[0])
		if err != nil || vol < 0 || vol > 100 {
			fmt.Fprintln(os.Stderr, "Volume must be 0–100")
			os.Exit(1)
		}
		if err := music.SetVolume(vol); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("♪  Volume set to %d%%\n", vol)
	},
}

// ── Seek / Shuffle / Repeat ──────────────────────────────────────────────────

var seekCmd = &cobra.Command{
	Use:   "seek <+seconds|-seconds>",
	Short: "Seek forward or backward by seconds",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		delta, err := parseSeekDelta(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := music.Seek(delta); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if delta >= 0 {
			fmt.Printf("⏩  Seeked forward %.0f second(s)\n", delta)
		} else {
			fmt.Printf("⏪  Seeked backward %.0f second(s)\n", -delta)
		}
	},
}

var shuffleCmd = &cobra.Command{
	Use:   "shuffle toggle",
	Short: "Toggle shuffle mode",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if args[0] != "toggle" {
			fmt.Fprintln(os.Stderr, `Usage: mu shuffle toggle`)
			os.Exit(1)
		}
		enabled, err := music.ToggleShuffle()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if enabled {
			fmt.Println("⇄ Shuffle on")
		} else {
			fmt.Println("⇄ Shuffle off")
		}
	},
}

var repeatCmd = &cobra.Command{
	Use:   "repeat",
	Short: "Cycle repeat mode: off → all → one",
	Run: func(cmd *cobra.Command, args []string) {
		mode, err := music.CycleRepeat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("↻ Repeat %s\n", mode)
	},
}

func parseSeekDelta(raw string) (float64, error) {
	if raw == "" || (raw[0] != '+' && raw[0] != '-') {
		return 0, fmt.Errorf("seek value must include a sign, for example +30 or -10")
	}
	delta, err := strconv.ParseFloat(raw, 64)
	if err != nil || delta == 0 {
		return 0, fmt.Errorf("seek value must be a non-zero signed number of seconds")
	}
	return delta, nil
}

// ── Playlist ──────────────────────────────────────────────────────────────────

var playlistCmd = &cobra.Command{
	Use:   "playlist",
	Short: "Manage and play playlists",
}

var playlistPlayCmd = &cobra.Command{
	Use:   "play <name>",
	Short: "Play a playlist by name",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := strings.Join(args, " ")
		fmt.Printf("Playing playlist: %s...\n", name)
		var err error
		if playlistShuffle {
			err = music.PlayPlaylistShuffled(name)
		} else {
			err = music.PlayPlaylist(name)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var playlistExportFormat string
var playlistExportOutput string

var playlistExportCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Export a playlist to M3U or JSON",
	Long:  "Export a Music.app playlist's tracks as M3U or JSON content. Use --format to choose (default: m3u). Use --output to write to a file instead of stdout.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		format := playlistExportFormat
		if format == "" {
			format = "m3u"
		}

		content, err := music.ExportPlaylist(name, format)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if playlistExportOutput != "" {
			if err := os.WriteFile(playlistExportOutput, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Exported playlist %q to %s\n", name, playlistExportOutput)
		} else {
			fmt.Print(content)
		}
	},
}

var playlistImportCmd = &cobra.Command{
	Use:   "import <name> <file>",
	Short: "Import a playlist from M3U or JSON file",
	Long:  "Import tracks from an M3U or JSON file into a Music.app playlist. The playlist is created if it does not already exist.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		filePath := args[1]

		if err := music.ImportPlaylist(name, filePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Imported tracks into playlist: %s\n", name)
	},
}

var playlistsCmd = &cobra.Command{
	Use:   "playlists",
	Short: "List playable playlists",
	Run: func(cmd *cobra.Command, args []string) {
		mine, others, err := music.GetCategorizedPlaylistInfos()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if playlistsJsonOutput {
			out := map[string][]music.PlaylistInfo{
				"mine":   mine,
				"others": others,
			}
			bz, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(bz))
			return
		}
		fmt.Print(formatPlaylistGroups(mine, others))
	},
}

func formatPlaylistGroups(mine, others []music.PlaylistInfo) string {
	var sb strings.Builder
	writeGroup := func(title string, playlists []music.PlaylistInfo) {
		if len(playlists) == 0 {
			return
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(title)
		sb.WriteString("\n")
		for _, p := range playlists {
			if p.TrackCount > 0 {
				sb.WriteString(fmt.Sprintf("  %s (%d tracks)\n", p.Name, p.TrackCount))
			} else {
				sb.WriteString(fmt.Sprintf("  %s\n", p.Name))
			}
		}
	}
	writeGroup("My Playlists:", mine)
	writeGroup("Apple Music & Shared:", others)
	if sb.Len() == 0 {
		return "No playlists found.\n"
	}
	return sb.String()
}

// ── Search ────────────────────────────────────────────────────────────────────

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search library by title, artist, or album",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")

		// Determine if any specific filters are active.
		hasFilters := searchArtist != "" || searchAlbum != "" || searchGenre != "" || searchYear > 0 || cmd.Flags().Changed("loved") || searchMinRating > 0

		var lovedPtr *bool
		if cmd.Flags().Changed("loved") {
			lovedPtr = &searchLoved
		}

		var results []music.TrackInfo
		var err error

		if hasFilters || query != "" {
			results, err = music.SearchFiltered(query, searchArtist, searchAlbum, searchGenre, searchYear, lovedPtr, searchMinRating, searchLimit)
		} else {
			fmt.Fprintln(os.Stderr, "Error: provide a search query or at least one filter (--artist, --album, --genre, --year, --loved, --min-rating)")
			os.Exit(1)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(results) == 0 {
			fmt.Println("No tracks found.")
			return
		}
		if searchJsonOutput {
			bz, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(bz))
			return
		}
		if hasFilters {
			fmt.Printf("%d result(s):\n\n", len(results))
		} else {
			fmt.Printf("%d result(s) for \"%s\":\n\n", len(results), query)
		}
		for i, t := range results {
			fmt.Printf("  [%d] %s — %s  (%s)\n", i+1, t.Title, t.Artist, t.Album)
		}
	},
}

// ── Love ──────────────────────────────────────────────────────────────────────

var loveCmd = &cobra.Command{
	Use:   "love",
	Short: "Toggle loved status for the current track",
	Run: func(cmd *cobra.Command, args []string) {
		loved, err := music.ToggleLove()
		if err != nil {
			// Streaming tracks may not support loved — surface a clear message
			fmt.Fprintf(os.Stderr, "Could not toggle loved: %v\n", err)
			os.Exit(1)
		}
		if loved {
			fmt.Println("♥  Loved!")
		} else {
			fmt.Println("♡  Removed from Loved")
		}
	},
}

// ── Stats ─────────────────────────────────────────────────────────────────────

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show play count, rating, and metadata for the current track",
	Run: func(cmd *cobra.Command, args []string) {
		stats, err := music.GetTrackStats()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if statsJsonOutput {
			bz, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(bz))
			return
		}
		loved := "No"
		if stats.Loved {
			loved = "♥ Yes"
		}
		stars := strings.Repeat("★", stats.Rating/20) + strings.Repeat("☆", 5-stats.Rating/20)
		dateAdded := stats.DateAdded
		if dateAdded == "" || dateAdded == "missing value" {
			dateAdded = "—"
		}
		fmt.Printf("🎵 %s\n👤 %s\n💿 %s\n\nPlay Count : %d\nRating     : %s (%d/100)\nLoved      : %s\nDate Added : %s\n",
			stats.Title, stats.Artist, stats.Album,
			stats.PlayCount, stars, stats.Rating, loved, dateAdded)
	},
}

// ── Share ─────────────────────────────────────────────────────────────────────

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Copy the current track name and artist to the clipboard",
	Run: func(cmd *cobra.Command, args []string) {
		info, err := music.NowPlaying()
		if err != nil || info.State == "stopped" {
			fmt.Fprintln(os.Stderr, "Nothing is playing.")
			os.Exit(1)
		}
		text := fmt.Sprintf("%s — %s", info.Title, info.Artist)
		if _, err := exec.LookPath("pbcopy"); err != nil {
			fmt.Fprintln(os.Stderr, "pbcopy not found (clipboard is macOS only)")
			fmt.Println(text)
			return
		}
		pb := exec.Command("pbcopy")
		pb.Stdin = strings.NewReader(text)
		if err := pb.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Clipboard error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Copied to clipboard: %s\n", text)
	},
}

// ── Queue ─────────────────────────────────────────────────────────────────────

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage the DJ queue or show upcoming tracks",
	Run: func(cmd *cobra.Command, args []string) {
		tracks, err := music.GetQueue()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if queueJsonOutput {
			bz, _ := json.MarshalIndent(tracks, "", "  ")
			fmt.Println(string(bz))
			return
		}
		if len(tracks) == 0 {
			fmt.Println("No upcoming tracks.")
			return
		}
		fmt.Println("Up Next:")
		fmt.Println()
		for i, t := range tracks {
			fmt.Printf("  [%d] %s — %s\n", i+1, t.Title, t.Artist)
		}
	},
}

var queueAddCmd = &cobra.Command{
	Use:   "add <query>",
	Short: "Search and add the first matching track to the DJ queue",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")
		desc, pos, err := music.DJAddToQueue(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Queued: %s (%d in queue)\n", desc, pos)
	},
}

var queueClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all tracks from the DJ queue",
	Run: func(cmd *cobra.Command, args []string) {
		if err := music.DJClearQueue(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DJ queue cleared.")
	},
}

var queueMoveCmd = &cobra.Command{
	Use:   "move <from> <to>",
	Short: "Move a track from one position to another in the DJ queue",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		from, err := strconv.Atoi(args[0])
		if err != nil || from <= 0 {
			fmt.Fprintln(os.Stderr, "from must be a positive integer")
			os.Exit(1)
		}
		to, err := strconv.Atoi(args[1])
		if err != nil || to <= 0 {
			fmt.Fprintln(os.Stderr, "to must be a positive integer")
			os.Exit(1)
		}
		if err := music.DJMoveQueueItem(from, to); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Moved item from position %d to %d.\n", from, to)
	},
}

// ── Sleep Timer ───────────────────────────────────────────────────────────────

var sleepCmd = &cobra.Command{
	Use:   "sleep <minutes>",
	Short: "Stop music after the given number of minutes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mins, err := strconv.Atoi(args[0])
		if err != nil || mins <= 0 {
			fmt.Fprintln(os.Stderr, "Please provide a positive number of minutes.")
			os.Exit(1)
		}

		deadline := time.Now().Add(time.Duration(mins) * time.Minute)
		fmt.Printf("⏰ Music will stop in %d minute(s). Press Ctrl+C to cancel.\n\n", mins)

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sig)

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sig:
				fmt.Println("\n🚫 Sleep timer cancelled.")
				return
			case <-ticker.C:
				remaining := time.Until(deadline)
				if remaining <= 0 {
					fmt.Print("\r                                    \r")
					_ = music.Stop()
					fmt.Println("⏹  Music stopped. Goodnight! 🌙")
					return
				}
				m := int(remaining.Minutes())
				s := int(remaining.Seconds()) % 60
				fmt.Printf("\r⏰ Stopping in %02d:%02d...  ", m, s)
			}
		}
	},
}

// ── Mini Mode ─────────────────────────────────────────────────────────────────

var miniCmd = &cobra.Command{
	Use:   "mini",
	Short: "Show a compact single-line now-playing display (great for tmux)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.RunMini(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// ── Doctor ────────────────────────────────────────────────────────────────────

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run system diagnostics",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking Muse installation...")
		fmt.Println()
		allOk := true

		check := func(label string, ok bool, detail string) {
			if ok {
				fmt.Printf("  %-32s ✅\n", label)
			} else {
				fmt.Printf("  %-32s ❌  %s\n", label, detail)
				allOk = false
			}
		}

		check("macOS detected", runtime.GOOS == "darwin", "Muse only runs on macOS")

		_, err := exec.LookPath("osascript")
		check("osascript available", err == nil, "osascript not found in PATH")

		running, _ := music.IsMusicRunning()
		if running {
			fmt.Printf("  %-32s ✅\n", "Music.app running")
		} else {
			fmt.Printf("  %-32s ⚠️   not running (launches automatically)\n", "Music.app running")
		}

		_, scriptErr := music.RunAppleScript(`tell application "Music" to get player state`)
		if scriptErr != nil {
			check("AppleScript control access", false,
				"Open System Settings → Privacy & Security → Automation and grant terminal access to Music.app")
		} else {
			check("AppleScript control access", true, "")
		}

		fmt.Printf("  %-32s ✅  mu completion bash|zsh|fish\n", "Shell completions")
		if exe, err := os.Executable(); err == nil {
			fmt.Printf("  %-32s ✅  %s\n", "Executable path", exe)
		}
		if path, err := exec.LookPath("mu"); err == nil {
			fmt.Printf("  %-32s ✅  %s\n", "mu found in PATH", path)
		} else {
			fmt.Printf("  %-32s ⚠️   install with make install or scripts/install.sh\n", "mu found in PATH")
		}
		if Version == "dev" {
			fmt.Printf("  %-32s ⚠️   dev build; release builds inject a version tag\n", "Build version")
		} else {
			fmt.Printf("  %-32s ✅  %s\n", "Build version", Version)
		}

		fmt.Println()
		if allOk {
			fmt.Println("🎉 All checks passed! Muse is ready.")
		} else {
			fmt.Println("⚠️  Some checks failed.")
			os.Exit(1)
		}
	},
}

// ── Version ───────────────────────────────────────────────────────────────────

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Muse version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Muse %s\n", Version)
	},
}

// ── MCP ───────────────────────────────────────────────────────────────────────

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the Muse MCP server over stdio",
	Run: func(cmd *cobra.Command, args []string) {
		mcp.StartServer()
	},
}

// ── Config ────────────────────────────────────────────────────────────────────

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage Muse configuration",
	Long:  "Manage Muse preferences stored in ~/.config/muse/config.json.\nSubcommands: get, set, list, path.",
	Run: func(cmd *cobra.Command, args []string) {
		// Default: show all config
		cmd.Help()
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		val, err := config.Get(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(val)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.Set(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s = %s\n", args[0], args[1])
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	Run: func(cmd *cobra.Command, args []string) {
		keys, vals := config.ListSorted()
		if len(keys) == 0 {
			fmt.Println("No config found.")
			return
		}
		maxLen := 0
		for _, k := range keys {
			if len(k) > maxLen {
				maxLen = len(k)
			}
		}
		for i, k := range keys {
			fmt.Printf("  %-*s = %s\n", maxLen, k, vals[i])
		}
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.Path())
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configPathCmd)
}
