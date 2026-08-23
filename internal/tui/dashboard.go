package tui

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/art"
	lyricslib "github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/lyrics"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/music"
)

// ── Message types ─────────────────────────────────────────────────────────────

type tickMsg time.Time
type discoTickMsg time.Time
type noopMsg struct{}

type statusMsg struct {
	info *music.TrackInfo
	err  error
}
type playlistsMsg struct {
	mine   []music.PlaylistInfo
	others []music.PlaylistInfo
	err    error
}
type searchMsg struct {
	tracks []music.TrackInfo
	err    error
}
type playlistTracksMsg struct {
	tracks []music.TrackInfo
	err    error
}
type lyricsMsg struct {
	lines []lyricslib.Line
	err   error
}
type queueMsg struct {
	tracks []music.TrackInfo
	err    error
}
type historyMsg struct {
	recentTracks []music.TrackInfo
	topTracks    []music.TrackStats
	libraryStats *music.LibraryStats
	err          error
}
type artMsg struct {
	rendered string
	width    int
	height   int
}

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	accentColor = lipgloss.Color("#FF2D55")
	subtleColor = lipgloss.Color("#8E8E93")
	cardBgColor = lipgloss.Color("#2C2C2E")
	textColor   = lipgloss.Color("#FFFFFF")
	redColor    = lipgloss.Color("#FF3B30")

	docStyle = lipgloss.NewStyle().Padding(1, 2).Foreground(textColor)

	cardStyle = lipgloss.NewStyle().
			Background(cardBgColor).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Width(60)

	titleStyle = lipgloss.NewStyle().Foreground(accentColor).Bold(true).MarginBottom(1)
	logoStyle  = lipgloss.NewStyle().Foreground(redColor).Bold(true)

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(accentColor).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().Foreground(textColor).Padding(0, 1)
	helpStyle   = lipgloss.NewStyle().Foreground(subtleColor).MarginTop(1)

	themeMu sync.Mutex
)

type uiTheme struct {
	accent lipgloss.Color
	subtle lipgloss.Color
	cardBg lipgloss.Color
	text   lipgloss.Color
	logo   lipgloss.Color
}

var normalTheme = uiTheme{
	accent: lipgloss.Color("#FF2D55"),
	subtle: lipgloss.Color("#8E8E93"),
	cardBg: lipgloss.Color("#2C2C2E"),
	text:   lipgloss.Color("#FFFFFF"),
	logo:   lipgloss.Color("#FF3B30"),
}

var discoThemes = []uiTheme{
	{accent: lipgloss.Color("#FF1744"), subtle: lipgloss.Color("#FFAB40"), cardBg: lipgloss.Color("#2B1828"), text: lipgloss.Color("#FFFFFF"), logo: lipgloss.Color("#FF3D00")},
	{accent: lipgloss.Color("#00E5FF"), subtle: lipgloss.Color("#EA80FC"), cardBg: lipgloss.Color("#11212A"), text: lipgloss.Color("#FFFFFF"), logo: lipgloss.Color("#00B8D4")},
	{accent: lipgloss.Color("#76FF03"), subtle: lipgloss.Color("#FFFF00"), cardBg: lipgloss.Color("#172412"), text: lipgloss.Color("#F7FFF0"), logo: lipgloss.Color("#64DD17")},
	{accent: lipgloss.Color("#D500F9"), subtle: lipgloss.Color("#40C4FF"), cardBg: lipgloss.Color("#241332"), text: lipgloss.Color("#FFFFFF"), logo: lipgloss.Color("#AA00FF")},
	{accent: lipgloss.Color("#FFD600"), subtle: lipgloss.Color("#FF4081"), cardBg: lipgloss.Color("#292312"), text: lipgloss.Color("#FFFBEA"), logo: lipgloss.Color("#FFEA00")},
}

func applyTheme(theme uiTheme) {
	themeMu.Lock()
	defer themeMu.Unlock()
	accentColor = theme.accent
	subtleColor = theme.subtle
	cardBgColor = theme.cardBg
	textColor = theme.text
	redColor = theme.logo

	docStyle = lipgloss.NewStyle().Padding(1, 2).Foreground(textColor)
	cardStyle = lipgloss.NewStyle().
		Background(cardBgColor).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Width(60)
	titleStyle = lipgloss.NewStyle().Foreground(accentColor).Bold(true).MarginBottom(1)
	logoStyle = lipgloss.NewStyle().Foreground(redColor).Bold(true)
	highlightStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(accentColor).
		Bold(true).
		Padding(0, 1)
	normalStyle = lipgloss.NewStyle().Foreground(textColor).Padding(0, 1)
	helpStyle = lipgloss.NewStyle().Foreground(subtleColor).MarginTop(1)
}

type musicActions struct {
	NowPlaying                             func() (*music.TrackInfo, error)
	GetCategorizedPlaylistInfos            func() ([]music.PlaylistInfo, []music.PlaylistInfo, error)
	GetPlaylistTracksByPersistentID        func(string) ([]music.TrackInfo, error)
	Search                                 func(string) ([]music.TrackInfo, error)
	GetQueue                               func() ([]music.TrackInfo, error)
	GetArtworkPath                         func() (string, error)
	RunAppleScript                         func(string) (string, error)
	Toggle                                 func() error
	Next                                   func() error
	Prev                                   func() error
	CycleRepeat                            func() (string, error)
	ToggleShuffle                          func() (bool, error)
	ToggleLove                             func() (bool, error)
	Seek                                   func(float64) error
	SetVolume                              func(int) error
	Play                                   func() error
	Pause                                  func() error
	PlayPlaylistByPersistentID             func(string) error
	PlayPlaylistShuffledByPersistentID     func(string) error
	PlayPlaylistTrackAtIndexByPersistentID func(string, int) error
	PlayTrackByName                        func(string) error
	GetRecentlyPlayed                      func(int) ([]music.TrackInfo, error)
	GetTopTracks                           func(int) ([]music.TrackStats, error)
	GetLibraryStats                        func() (*music.LibraryStats, error)
}

var musicAPI = musicActions{
	NowPlaying:                             music.NowPlaying,
	GetCategorizedPlaylistInfos:            music.GetCategorizedPlaylistInfos,
	GetPlaylistTracksByPersistentID:        music.GetPlaylistTracksByPersistentID,
	Search:                                 music.Search,
	GetQueue:                               music.GetQueue,
	GetArtworkPath:                         music.GetArtworkPath,
	RunAppleScript:                         music.RunAppleScript,
	Toggle:                                 music.Toggle,
	Next:                                   music.Next,
	Prev:                                   music.Prev,
	CycleRepeat:                            music.CycleRepeat,
	ToggleShuffle:                          music.ToggleShuffle,
	ToggleLove:                             music.ToggleLove,
	Seek:                                   music.Seek,
	SetVolume:                              music.SetVolume,
	Play:                                   music.Play,
	Pause:                                  music.Pause,
	PlayPlaylistByPersistentID:             music.PlayPlaylistByPersistentID,
	PlayPlaylistShuffledByPersistentID:     music.PlayPlaylistShuffledByPersistentID,
	PlayPlaylistTrackAtIndexByPersistentID: music.PlayPlaylistTrackAtIndexByPersistentID,
	PlayTrackByName:                        music.PlayTrackByName,
	GetRecentlyPlayed:                      music.GetRecentlyPlayed,
	GetTopTracks:                           music.GetTopTracks,
	GetLibraryStats:                        music.GetLibraryStats,
}

// ── State machine ─────────────────────────────────────────────────────────────

type menuState int

const (
	stateMainMenu menuState = iota
	statePlaylists
	statePlaylistDetail
	stateSearchInput
	stateSearchResults
	stateLyrics
	stateQueue
	stateHistory
)

// ── Model ─────────────────────────────────────────────────────────────────────

type model struct {
	state        menuState
	currentTrack *music.TrackInfo
	errorMsg     string
	discoMode    bool
	discoPhase   int
	// Main menu grid
	activeColumn int
	activeRow    int
	// Playlists
	myPlaylists          []music.PlaylistInfo
	otherPlaylists       []music.PlaylistInfo
	playlistIndex        int
	selectedPlaylist     music.PlaylistInfo
	playlistTracks       []music.TrackInfo
	playlistDetailIndex  int
	activePlaylistID     string
	activePlaylistTracks []music.TrackInfo
	activePlaylistIndex  int
	// Search
	searchInput   textinput.Model
	searchResults []music.TrackInfo
	searchIndex   int
	loading       bool
	// Lyrics
	lyricsLines   []lyricslib.Line
	lyricsLoading bool
	lyricsErr     string
	// Queue
	queueTracks  []music.TrackInfo
	queueLoading bool
	queueIndex   int
	// Album art
	artRendered    string
	artWidth       int
	artHeight      int
	lastTrackTitle string
	// History & Stats
	historyRecentTracks []music.TrackInfo
	historyTopTracks    []music.TrackStats
	historyLibraryStats *music.LibraryStats
	historyLoading      bool
	historyError        string
	// Window
	width  int
	height int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Search for a song or artist..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 30

	return model{
		state:        stateMainMenu,
		searchInput:  ti,
		currentTrack: &music.TrackInfo{State: "stopped"},
	}
}

// ── Commands ──────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(m.updateStatusCmd(), tickCmd(), discoTickCmd(700*time.Millisecond))
}

func (m model) updateStatusCmd() tea.Cmd {
	return func() tea.Msg {
		info, err := musicAPI.NowPlaying()
		return statusMsg{info: info, err: err}
	}
}

func (m model) fetchPlaylistsCmd() tea.Cmd {
	return func() tea.Msg {
		mine, others, err := musicAPI.GetCategorizedPlaylistInfos()
		return playlistsMsg{mine: mine, others: others, err: err}
	}
}

func (m model) fetchPlaylistTracksCmd(playlist music.PlaylistInfo) tea.Cmd {
	return func() tea.Msg {
		tracks, err := musicAPI.GetPlaylistTracksByPersistentID(playlist.PersistentID)
		return playlistTracksMsg{tracks: tracks, err: err}
	}
}

func (m model) executeSearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		res, err := musicAPI.Search(query)
		return searchMsg{tracks: res, err: err}
	}
}

func fetchLyricsCmd(title, artist, album string, duration float64) tea.Cmd {
	return func() tea.Msg {
		lines, err := lyricslib.Fetch(title, artist, album, duration)
		return lyricsMsg{lines: lines, err: err}
	}
}

func fetchQueueCmd() tea.Cmd {
	return func() tea.Msg {
		tracks, err := musicAPI.GetQueue()
		return queueMsg{tracks: tracks, err: err}
	}
}

func fetchHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		recent, err1 := musicAPI.GetRecentlyPlayed(25)
		top, err2 := musicAPI.GetTopTracks(15)
		stats, err3 := musicAPI.GetLibraryStats()
		if err1 != nil {
			return historyMsg{err: err1}
		}
		if err2 != nil {
			return historyMsg{err: err2}
		}
		if err3 != nil {
			return historyMsg{err: err3}
		}
		return historyMsg{recentTracks: recent, topTracks: top, libraryStats: stats}
	}
}

func fetchArtCmd(width, height int) tea.Cmd {
	return func() tea.Msg {
		path, err := musicAPI.GetArtworkPath()
		if err != nil || path == "" {
			return artMsg{}
		}
		rendered, err := art.Render(path, width, height)
		// Clean up the temp file created by mktemp (best-effort).
		if path != "" && path != "/tmp/muse_art.jpg" {
			_ = os.Remove(path)
		}
		if err != nil {
			return artMsg{}
		}
		return artMsg{rendered: rendered, width: width, height: height}
	}
}

func sendNotificationCmd(title, artist string) tea.Cmd {
	return func() tea.Msg {
		escapeNotif := func(s string) string {
			s = strings.ReplaceAll(s, "\\", "\\\\")
			s = strings.ReplaceAll(s, "\"", "\\\"")
			s = strings.ReplaceAll(s, "\n", " ")
			s = strings.ReplaceAll(s, "\r", " ")
			return s
		}
		safeTitle := escapeNotif(title)
		safeArtist := escapeNotif(artist)
		script := fmt.Sprintf(`display notification "%s" with title "♫ Now Playing" subtitle "%s"`, safeTitle, safeArtist)
		musicAPI.RunAppleScript(script)
		return noopMsg{}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func discoTickCmd(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg { return discoTickMsg(t) })
}

func discoNowCmd() tea.Cmd {
	return func() tea.Msg { return discoTickMsg(time.Now()) }
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state != stateSearchInput && m.state != stateSearchResults {
				return m, tea.Quit
			}
		case "esc":
			switch m.state {
			case statePlaylistDetail:
				m.state = statePlaylists
				m.errorMsg = ""
				return m, nil
			case statePlaylists, stateSearchInput, stateSearchResults, stateLyrics, stateQueue, stateHistory:
				m.state = stateMainMenu
				m.errorMsg = ""
				return m, nil
			}
		case " ", "space":
			if m.state != stateSearchInput {
				return m.withControlError(musicAPI.Toggle())
			}
		case "n":
			if m.state != stateSearchInput {
				return m.playNextTrack()
			}
		case "p":
			if m.state != stateSearchInput {
				return m.playPreviousTrack()
			}
		case "r":
			if m.state != stateSearchInput {
				_, err := musicAPI.CycleRepeat()
				return m.withControlError(err)
			}
		case "s":
			if m.state != stateSearchInput {
				_, err := musicAPI.ToggleShuffle()
				return m.withControlError(err)
			}
		case "l":
			if m.state != stateSearchInput {
				_, err := musicAPI.ToggleLove()
				return m.withControlError(err)
			}
		case "d":
			if m.state != stateSearchInput {
				m.discoMode = !m.discoMode
				if !m.discoMode {
					m.discoPhase = 0
				}
				return m, discoNowCmd()
			}
		case "[":
			if m.state != stateSearchInput {
				return m.withControlError(musicAPI.Seek(-10))
			}
		case "]":
			if m.state != stateSearchInput {
				return m.withControlError(musicAPI.Seek(10))
			}
		case "+", "=":
			if m.state != stateSearchInput {
				newVol := m.currentTrack.Volume + 5
				if newVol > 100 {
					newVol = 100
				}
				return m.withControlError(musicAPI.SetVolume(newVol))
			}
		case "-":
			if m.state != stateSearchInput {
				newVol := m.currentTrack.Volume - 5
				if newVol < 0 {
					newVol = 0
				}
				return m.withControlError(musicAPI.SetVolume(newVol))
			}
		}

		// State-specific key handling
		switch m.state {
		case stateMainMenu:
			columns := menuColumns()
			switch msg.String() {
			case "left", "h":
				if m.activeColumn > 0 {
					m.activeColumn--
					if m.activeRow >= len(columns[m.activeColumn]) {
						m.activeRow = len(columns[m.activeColumn]) - 1
					}
				}
			case "right", "l":
				if m.activeColumn < len(columns)-1 {
					m.activeColumn++
					if m.activeRow >= len(columns[m.activeColumn]) {
						m.activeRow = len(columns[m.activeColumn]) - 1
					}
				}
			case "up", "k":
				if m.activeRow > 0 {
					m.activeRow--
				}
			case "down", "j":
				if m.activeRow < len(columns[m.activeColumn])-1 {
					m.activeRow++
				}
			case "enter":
				return m.handleMenuSelect(columns[m.activeColumn][m.activeRow])
			}

		case statePlaylists:
			all := append(m.myPlaylists, m.otherPlaylists...)
			switch msg.String() {
			case "up", "k":
				if m.playlistIndex > 0 {
					m.playlistIndex--
				}
			case "down", "j":
				if m.playlistIndex < len(all)-1 {
					m.playlistIndex++
				}
			case "enter":
				if len(all) > 0 {
					playlist := all[m.playlistIndex]
					m.selectedPlaylist = playlist
					m.playlistDetailIndex = 0
					m.playlistTracks = nil
					m.state = statePlaylistDetail
					return m, m.fetchPlaylistTracksCmd(playlist)
				}
			}

		case statePlaylistDetail:
			switch msg.String() {
			case "up", "k":
				if m.playlistDetailIndex > 0 {
					m.playlistDetailIndex--
				}
			case "down", "j":
				// 0=Play Normally, 1=Play Shuffled, 2..n+1=tracks
				maxIndex := 1
				if len(m.playlistTracks) > 0 {
					maxIndex = 1 + len(m.playlistTracks)
				}
				if m.playlistDetailIndex < maxIndex {
					m.playlistDetailIndex++
				}
			case "enter":
				switch m.playlistDetailIndex {
				case 0:
					if err := musicAPI.PlayPlaylistByPersistentID(m.selectedPlaylist.PersistentID); err != nil {
						return m.withControlError(err)
					}
					m = m.clearPlaylistTrackContext()
				case 1:
					if err := musicAPI.PlayPlaylistShuffledByPersistentID(m.selectedPlaylist.PersistentID); err != nil {
						return m.withControlError(err)
					}
					m = m.clearPlaylistTrackContext()
				default:
					idx := m.playlistDetailIndex - 2
					if idx >= 0 && idx < len(m.playlistTracks) {
						if err := musicAPI.PlayPlaylistTrackAtIndexByPersistentID(m.selectedPlaylist.PersistentID, idx); err != nil {
							return m.withControlError(err)
						}
						m.activePlaylistID = m.selectedPlaylist.PersistentID
						m.activePlaylistTracks = append([]music.TrackInfo(nil), m.playlistTracks...)
						m.activePlaylistIndex = idx
					}
				}
				m.state = stateMainMenu
				return m, m.updateStatusCmd()
			}

		case stateSearchInput:
			if msg.String() == "enter" {
				if q := m.searchInput.Value(); q != "" {
					m.state = stateSearchResults
					m.loading = true
					m.searchResults = nil
					return m, m.executeSearchCmd(q)
				}
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)

		case stateSearchResults:
			switch msg.String() {
			case "up", "k":
				if m.searchIndex > 0 {
					m.searchIndex--
				}
			case "down", "j":
				if m.searchIndex < len(m.searchResults)-1 {
					m.searchIndex++
				}
			case "enter":
				if len(m.searchResults) > 0 {
					if err := musicAPI.PlayTrackByName(m.searchResults[m.searchIndex].Title); err != nil {
						return m.withControlError(err)
					}
					m = m.clearPlaylistTrackContext()
					m.state = stateMainMenu
					return m, m.updateStatusCmd()
				}
			}

		case stateLyrics:
			switch msg.String() {
			case "up", "k":
				// lyrics view is read-only / auto-scrolling; no cursor
			case "down", "j":
			}

		case stateQueue:
			switch msg.String() {
			case "up", "k":
				if m.queueIndex > 0 {
					m.queueIndex--
				}
			case "down", "j":
				if m.queueIndex < len(m.queueTracks)-1 {
					m.queueIndex++
				}
			case "enter":
				if len(m.queueTracks) > 0 {
					if err := musicAPI.PlayTrackByName(m.queueTracks[m.queueIndex].Title); err != nil {
						return m.withControlError(err)
					}
					m = m.clearPlaylistTrackContext()
					m.state = stateMainMenu
					return m, m.updateStatusCmd()
				}
			}

		case stateHistory:
			// history view is read-only; no cursor interaction needed
		}

	// ── Data messages ────────────────────────────────────────────────────────

	case statusMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			trackKey := ""
			if msg.info != nil {
				trackKey = msg.info.Title + "\x1f" + msg.info.Artist + "\x1f" + msg.info.Album
			}
			lastKey := m.lastTrackTitle
			// Backward-compat: early builds stored only title in lastTrackTitle.
			// Treat a bare-title key as matching if the current track's title matches.
			isSameTrack := trackKey != "" && (trackKey == lastKey || (msg.info != nil && msg.info.Title == lastKey))
			if msg.info != nil && msg.info.Title != "" && !isSameTrack {
				// Track changed — fetch new art, notify, clear stale lyrics
				m.lastTrackTitle = trackKey
				m.artRendered = ""
				m.artWidth = 0
				m.artHeight = 0
				m.lyricsLines = nil
				m.lyricsErr = ""
				artW, artH := albumArtSize(m.cardWidth())
				cmds = append(cmds, fetchArtCmd(artW, artH), sendNotificationCmd(msg.info.Title, msg.info.Artist))
				if m.state == stateLyrics {
					m.lyricsLoading = true
					cmds = append(cmds, fetchLyricsCmd(msg.info.Title, msg.info.Artist, msg.info.Album, msg.info.Duration))
				}

				// Synchronize activePlaylistIndex if we have playlist context
				if m.hasPlaylistTrackContext() {
					found := false
					for idx, track := range m.activePlaylistTracks {
						if track.Title == msg.info.Title && track.Artist == msg.info.Artist && track.Album == msg.info.Album {
							m.activePlaylistIndex = idx
							found = true
							break
						}
					}
					if !found {
						m = m.clearPlaylistTrackContext()
					}
				}
			}
			m.currentTrack = msg.info
			m.errorMsg = ""
		}

	case playlistsMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			m.myPlaylists = msg.mine
			m.otherPlaylists = msg.others
			m.playlistIndex = 0
			m.errorMsg = ""
		}

	case searchMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			m.searchResults = msg.tracks
			m.searchIndex = 0
			m.errorMsg = ""
		}

	case playlistTracksMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			m.playlistTracks = msg.tracks
			m.playlistDetailIndex = 0
			m.errorMsg = ""
		}

	case lyricsMsg:
		m.lyricsLoading = false
		if msg.err != nil {
			m.lyricsErr = msg.err.Error()
		} else {
			m.lyricsLines = msg.lines
			m.lyricsErr = ""
		}

	case queueMsg:
		m.queueLoading = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			m.queueTracks = msg.tracks
			m.queueIndex = 0
		}

	case historyMsg:
		m.historyLoading = false
		if msg.err != nil {
			m.historyError = msg.err.Error()
		} else {
			m.historyRecentTracks = msg.recentTracks
			m.historyTopTracks = msg.topTracks
			m.historyLibraryStats = msg.libraryStats
			m.historyError = ""
		}

	case artMsg:
		m.artRendered = msg.rendered
		m.artWidth = msg.width
		m.artHeight = msg.height

	case noopMsg:
		// fire-and-forget cmds (notifications) return this; nothing to do

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.currentTrack != nil && m.currentTrack.Title != "" {
			artW, artH := albumArtSize(m.cardWidth())
			if artW != m.artWidth || artH != m.artHeight {
				m.artRendered = ""
				return m, tea.Batch(tea.ClearScreen, fetchArtCmd(artW, artH))
			}
		}
		return m, tea.ClearScreen

	case tickMsg:
		cmds = append(cmds, m.updateStatusCmd(), tickCmd())

	case discoTickMsg:
		if m.discoActive() {
			m.discoPhase++
			cmds = append(cmds, discoTickCmd(140*time.Millisecond))
		} else {
			cmds = append(cmds, discoTickCmd(700*time.Millisecond))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) hasPlaylistTrackContext() bool {
	return m.activePlaylistID != "" &&
		m.activePlaylistIndex >= 0 &&
		m.activePlaylistIndex < len(m.activePlaylistTracks)
}

func (m model) clearPlaylistTrackContext() model {
	m.activePlaylistID = ""
	m.activePlaylistTracks = nil
	m.activePlaylistIndex = 0
	return m
}

func (m model) withControlError(err error) (tea.Model, tea.Cmd) {
	if err != nil {
		m.errorMsg = err.Error()
		return m, m.updateStatusCmd()
	}
	m.errorMsg = ""
	return m, m.updateStatusCmd()
}

func (m model) playPlaylistTrackAt(index int) (tea.Model, tea.Cmd) {
	if m.activePlaylistID == "" || index < 0 || index >= len(m.activePlaylistTracks) {
		return m, m.updateStatusCmd()
	}
	if err := musicAPI.PlayPlaylistTrackAtIndexByPersistentID(m.activePlaylistID, index); err != nil {
		return m.withControlError(err)
	}
	m.activePlaylistIndex = index
	return m.withControlError(nil)
}

func (m model) playNextTrack() (tea.Model, tea.Cmd) {
	if m.hasPlaylistTrackContext() {
		nextIndex := m.activePlaylistIndex + 1
		if nextIndex >= len(m.activePlaylistTracks) {
			nextIndex = 0
		}
		return m.playPlaylistTrackAt(nextIndex)
	}
	return m.withControlError(musicAPI.Next())
}

func (m model) playPreviousTrack() (tea.Model, tea.Cmd) {
	if m.hasPlaylistTrackContext() {
		prevIndex := m.activePlaylistIndex - 1
		if prevIndex < 0 {
			prevIndex = len(m.activePlaylistTracks) - 1
		}
		return m.playPlaylistTrackAt(prevIndex)
	}
	return m.withControlError(musicAPI.Prev())
}

// menuColumns defines the main menu grid.
func menuColumns() [][]string {
	return [][]string{
		{"Library Playlists", "Search Library", "Lyrics", "Up Next", "History & Stats"},
		{"Play", "Pause", "Next", "Previous"},
	}
}

func (m model) handleMenuSelect(item string) (tea.Model, tea.Cmd) {
	switch item {
	case "Library Playlists":
		m.state = statePlaylists
		return m, m.fetchPlaylistsCmd()
	case "Search Library":
		m.state = stateSearchInput
		m.searchInput.SetValue("")
		m.searchInput.Focus()
	case "Lyrics":
		m.state = stateLyrics
		// Fetch if not yet loaded, or retry after a previous error
		needsFetch := (m.lyricsLines == nil || m.lyricsErr != "") && !m.lyricsLoading
		if needsFetch && m.currentTrack != nil && m.currentTrack.Title != "" {
			m.lyricsLoading = true
			m.lyricsErr = ""
			return m, fetchLyricsCmd(m.currentTrack.Title, m.currentTrack.Artist, m.currentTrack.Album, m.currentTrack.Duration)
		}
	case "Up Next":
		m.state = stateQueue
		m.queueLoading = true
		m.queueTracks = nil
		return m, fetchQueueCmd()
	case "History & Stats":
		m.state = stateHistory
		m.historyLoading = true
		m.historyRecentTracks = nil
		m.historyTopTracks = nil
		m.historyLibraryStats = nil
		m.historyError = ""
		return m, fetchHistoryCmd()
	case "Play":
		return m.withControlError(musicAPI.Play())
	case "Pause":
		return m.withControlError(musicAPI.Pause())
	case "Next":
		return m.playNextTrack()
	case "Previous":
		return m.playPreviousTrack()
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

const asciiLogo = `███╗   ███╗██╗   ██╗███████╗███████╗
████╗ ████║██║   ██║██╔════╝██╔════╝
██╔████╔██║██║   ██║███████╗█████╗  
██║╚██╔╝██║██║   ██║╚════██║██╔══╝  
██║ ╚═╝ ██║╚██████╔╝███████║███████╗
╚═╝     ╚═╝ ╚═════╝ ╚══════╝╚══════╝`

func (m model) cardWidth() int {
	cardWidth := 60
	if m.width > 0 {
		cardWidth = m.width - 6
		if cardWidth < 30 {
			cardWidth = 30
		}
		if cardWidth > 112 {
			cardWidth = 112
		}
		// Never exceed the available width.
		if cardWidth > m.width-2 {
			cardWidth = m.width - 2
			if cardWidth < 30 {
				cardWidth = 30
			}
		}
	}
	return cardWidth
}

func (m model) discoActive() bool {
	return m.discoMode && m.currentTrack != nil && m.currentTrack.State == "playing"
}

func (m model) currentTheme() uiTheme {
	if m.discoActive() && len(discoThemes) > 0 {
		return discoThemes[m.discoPhase%len(discoThemes)]
	}
	return normalTheme
}

func albumArtSize(cardWidth int) (int, int) {
	switch {
	case cardWidth >= 100:
		return 34, 17
	case cardWidth >= 82:
		return 28, 14
	default:
		return 22, 11
	}
}

func (m model) View() string {
	applyTheme(m.currentTheme())

	var s strings.Builder

	s.WriteString(logoStyle.Render(asciiLogo))
	s.WriteString("\n\n")

	cardWidth := m.cardWidth()

	// ── Now Playing card ──────────────────────────────────────────────────────
	s.WriteString(cardStyle.Copy().Width(cardWidth).Render(m.renderNowPlaying(cardWidth)))
	s.WriteString("\n\n")

	if m.errorMsg != "" {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Render("Error: "+friendlyError(m.errorMsg)) + "\n\n")
	}

	// ── Content panel ─────────────────────────────────────────────────────────
	switch m.state {
	case stateMainMenu:
		s.WriteString(m.renderMainMenu(cardWidth))
	case statePlaylists:
		s.WriteString(m.renderPlaylists(cardWidth))
	case statePlaylistDetail:
		s.WriteString(m.renderPlaylistDetail(cardWidth))
	case stateSearchInput:
		s.WriteString(m.renderSearchInput(cardWidth))
	case stateSearchResults:
		s.WriteString(m.renderSearchResults(cardWidth))
	case stateLyrics:
		s.WriteString(m.renderLyrics(cardWidth))
	case stateQueue:
		s.WriteString(m.renderQueue(cardWidth))
	case stateHistory:
		s.WriteString(m.renderHistory(cardWidth))
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render(
		"↑/↓/←/→  Navigate  •  Enter Select  •  Space Toggle  •  n Next  •  p Prev  " +
			"•  l Love  •  d Disco  •  s Shuffle  •  r Repeat  •  [/] Seek  •  +/- Vol  •  q Quit",
	))

	return docStyle.Render(s.String())
}

// renderNowPlaying builds the content for the now-playing card.
func (m model) renderNowPlaying(cardWidth int) string {
	if m.currentTrack == nil || m.currentTrack.State == "stopped" {
		return "⏹  Music stopped\n\nOpen Music.app or press Play to start playback."
	}

	// ── Progress bar ──────────────────────────────────────────────────────────
	var progressBar string
	if m.currentTrack.Duration > 0 {
		totalW := cardWidth - 20
		if totalW < 15 {
			totalW = 15
		}
		pct := m.currentTrack.Position / m.currentTrack.Duration
		filled := int(pct * float64(totalW))
		if filled > totalW {
			filled = totalW
		}
		rem := totalW - filled

		var elapsed, head, remaining string
		if filled > 0 {
			elapsed = lipgloss.NewStyle().Foreground(accentColor).Render(strings.Repeat("━", filled-1))
			head = lipgloss.NewStyle().Foreground(accentColor).Render("●")
		} else {
			head = lipgloss.NewStyle().Foreground(subtleColor).Render("○")
		}
		remaining = lipgloss.NewStyle().Foreground(subtleColor).Render(strings.Repeat("─", rem))

		progressBar = fmt.Sprintf("%02d:%02d  %s%s%s  %02d:%02d",
			int(m.currentTrack.Position)/60, int(m.currentTrack.Position)%60,
			elapsed, head, remaining,
			int(m.currentTrack.Duration)/60, int(m.currentTrack.Duration)%60)
	}

	// ── Disc / album art ──────────────────────────────────────────────────────
	discStyle := lipgloss.NewStyle().Foreground(accentColor).PaddingRight(2).Align(lipgloss.Center)

	var discContent string
	discWidth := 22
	if m.artRendered != "" {
		discContent = m.artRendered
		discWidth = m.artWidth
	} else {
		spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinIdx := time.Now().UnixMilli() / 100 % int64(len(spinners))
		if m.currentTrack.State == "playing" {
			sp := spinners[spinIdx]
			discContent = "  ╭───────────────────╮\n" +
				"  │                   │\n" +
				"  │   ╭───────────╮   │\n" +
				"  │   │           │   │\n" +
				"  │   │  ╭─────╮  │   │\n" +
				sp + "  │   │  │  ⦿  │  │   │\n" +
				"  │   │  ╰─────╯  │   │\n" +
				"  │   │           │   │\n" +
				"  │   ╰───────────╯   │\n" +
				"  │                   │\n" +
				"  ╰───────────────────╯"
		} else {
			discContent = "  ╭───────────────────╮\n" +
				"  │                   │\n" +
				"  │   ╭───────────╮   │\n" +
				"  │   │           │   │\n" +
				"  │   │  ╭─────╮  │   │\n" +
				"  │   │  │  ⦿  │  │   │\n" +
				"  │   │  ╰─────╯  │   │\n" +
				"  │   │           │   │\n" +
				"  │   ╰───────────╯   │\n" +
				"  │                   │\n" +
				"  ╰───────────────────╯"
		}
	}

	// ── Right column ──────────────────────────────────────────────────────────
	// Album art uses truecolor half-blocks. The right column accounts for the
	// actual art width so wide terminals can render sharper covers.
	rightW := cardWidth - discWidth - 4
	if rightW < 25 {
		rightW = 25
	}

	// Loved heart
	loveStr := ""
	if m.currentTrack.Loved {
		loveStr = "  " + lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("♥")
	}

	// Shuffle / repeat indicators
	shuffleStr := lipgloss.NewStyle().Foreground(subtleColor).Render("⇄ Off")
	if m.currentTrack.Shuffle {
		shuffleStr = lipgloss.NewStyle().Foreground(accentColor).Render("⇄ On")
	}
	repeatStr := lipgloss.NewStyle().Foreground(subtleColor).Render("↻ Off")
	switch m.currentTrack.Repeat {
	case "all", "yes":
		repeatStr = lipgloss.NewStyle().Foreground(accentColor).Render("↻ All")
	case "one":
		repeatStr = lipgloss.NewStyle().Foreground(accentColor).Render("↻ One")
	}

	// Playback state label
	var stateLabel string
	if m.currentTrack.State == "playing" {
		stateLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#34C759")).Bold(true).Render("● Playing")
	} else {
		stateLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9500")).Bold(true).Render("○ Paused")
	}

	volStr := lipgloss.NewStyle().Foreground(subtleColor).Render(fmt.Sprintf("♪ %d%%", m.currentTrack.Volume))
	discoStr := lipgloss.NewStyle().Foreground(subtleColor).Render("✦ Off")
	if m.discoMode {
		discoStr = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("✦ Disco")
	}
	statusRow := fmt.Sprintf("%s   %s   %s   %s   %s%s", stateLabel, volStr, discoStr, shuffleStr, repeatStr, loveStr)

	// Big 3-line block title, wrapped to fit the column
	bigStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	titleBlock := renderBigTitle(m.currentTrack.Title, rightW)

	rightContent := fmt.Sprintf("%s\n\n%s\n%s\n\n%s",
		bigStyle.Render(titleBlock),
		lipgloss.NewStyle().Foreground(textColor).Bold(true).Width(rightW).MaxWidth(rightW).Render(m.currentTrack.Artist),
		lipgloss.NewStyle().Foreground(subtleColor).Italic(true).Width(rightW).MaxWidth(rightW).Render(m.currentTrack.Album),
		lipgloss.NewStyle().Width(rightW).MaxWidth(rightW).Render(statusRow),
	)

	card := lipgloss.JoinHorizontal(lipgloss.Top,
		discStyle.Render(discContent),
		lipgloss.NewStyle().Width(rightW).Render(rightContent),
	)

	if progressBar != "" {
		card = card + "\n\n" + progressBar
	}
	return card
}

func friendlyError(msg string) string {
	if strings.Contains(strings.ToLower(msg), "not authorized") ||
		strings.Contains(strings.ToLower(msg), "not permitted") ||
		strings.Contains(strings.ToLower(msg), "privacy") ||
		strings.Contains(strings.ToLower(msg), "automation") {
		return "AppleScript permission is blocked. Grant your terminal Automation access to Music.app in System Settings."
	}
	return msg
}

// ── Panel renderers ───────────────────────────────────────────────────────────

func (m model) renderMainMenu(cardWidth int) string {
	columns := menuColumns()

	// Width() in lipgloss is the OUTER box dimension (border + padding + content).
	// Padding(1,1): 1 top/bottom, 1 left/right  → horizontal overhead = 2(border) + 2(pad) = 4
	// Content area = colWidth - 4
	// Longest item: "Library Playlists" = 17 chars
	// Item render: "> 📂  Library Playlists" ≈ 24 chars (generous for emoji double-width uncertainty)
	// So colWidth must be >= 24 + 4 = 28. Use 36 minimum for comfort.
	icons := map[string]string{
		"Library Playlists": "📂",
		"Search Library":    "🔍",
		"Lyrics":            "🎵",
		"Up Next":           "⏭",
		"History & Stats":   "📊",
		"Play":              "▶ ",
		"Pause":             "⏸ ",
		"Next":              "→ ",
		"Previous":          "← ",
	}

	renderItem := func(item string, col, row int) string {
		icon := icons[item]
		sel := m.activeColumn == col && m.activeRow == row
		// Both branches same length so the box renders consistently
		if sel {
			return highlightStyle.Render("> " + icon + " " + item)
		}
		return normalStyle.Render("  " + icon + " " + item)
	}

	// Two boxes fill the same total width as the now-playing card.
	// Gap between boxes = 2 chars, so each box = (cardWidth - 2) / 2.
	colWidth := (cardWidth - 2) / 2
	if colWidth < 30 {
		colWidth = 30
	}

	borderColor := func(col int) lipgloss.Color {
		if m.activeColumn == col {
			return accentColor
		}
		return subtleColor
	}

	padTo := func(items string, n int) string {
		lines := strings.Split(strings.TrimSuffix(items, "\n"), "\n")
		for len(lines) < n {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	buildCol := func(idx int, header string) string {
		var sb strings.Builder
		for i, item := range columns[idx] {
			sb.WriteString(renderItem(item, idx, i) + "\n")
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor(idx)).
			Padding(1, 1).
			Width(colWidth).
			Render(header + "\n\n" + padTo(sb.String(), 4))
	}

	nav := buildCol(0, "NAVIGATION")
	ctrl := buildCol(1, "CONTROLS")

	return lipgloss.JoinHorizontal(lipgloss.Top, nav, "  ", ctrl) + "\n"
}

func (m model) renderPlaylists(cardWidth int) string {
	var sb strings.Builder

	notLoaded := m.myPlaylists == nil && m.otherPlaylists == nil
	if notLoaded {
		sb.WriteString(helpStyle.Render("  Loading playlists...") + "\n")
	} else {
		sectionHeader := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
		dim := lipgloss.NewStyle().Foreground(subtleColor)

		// Combined index offset so playlistIndex spans both slices
		renderList := func(items []music.PlaylistInfo, offset int) {
			for i, playlist := range items {
				idx := offset + i
				label := playlist.Name
				if playlist.TrackCount > 0 {
					label = fmt.Sprintf("%s  (%d)", playlist.Name, playlist.TrackCount)
				}
				if idx == m.playlistIndex {
					sb.WriteString(highlightStyle.Render("> "+label) + "\n")
				} else {
					sb.WriteString(normalStyle.Render("  "+label) + "\n")
				}
			}
		}

		if len(m.myPlaylists) > 0 {
			sb.WriteString(sectionHeader.Render("  MY PLAYLISTS") + "\n")
			renderList(m.myPlaylists, 0)
		}

		if len(m.otherPlaylists) > 0 {
			if len(m.myPlaylists) > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(sectionHeader.Render("  APPLE MUSIC & SHARED") + "\n")
			sb.WriteString(dim.Render("  ─────────────────────────") + "\n")
			renderList(m.otherPlaylists, len(m.myPlaylists))
		}

		if len(m.myPlaylists) == 0 && len(m.otherPlaylists) == 0 {
			sb.WriteString(normalStyle.Render("  No playlists found.") + "\n")
		}
	}

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
		Padding(0, 1).Width(cardWidth).Render("📂 PLAYLISTS (Esc to go back)\n\n" + sb.String())
	return box + "\n"
}

func (m model) renderPlaylistDetail(cardWidth int) string {
	var sb strings.Builder
	if m.playlistDetailIndex == 0 {
		sb.WriteString(highlightStyle.Render("> ▶  Play Normally") + "\n")
	} else {
		sb.WriteString(normalStyle.Render("  ▶  Play Normally") + "\n")
	}
	if m.playlistDetailIndex == 1 {
		sb.WriteString(highlightStyle.Render("> ⇄  Play Shuffled") + "\n\n")
	} else {
		sb.WriteString(normalStyle.Render("  ⇄  Play Shuffled") + "\n\n")
	}

	sb.WriteString("Tracks:\n────────────────────────\n")
	if m.playlistTracks == nil {
		sb.WriteString(helpStyle.Render("  Loading tracks...") + "\n")
	} else if len(m.playlistTracks) == 0 {
		sb.WriteString(normalStyle.Render("  No tracks in this playlist.") + "\n")
	} else {
		const maxVisible = 12
		cur := m.playlistDetailIndex - 2
		if cur < 0 {
			cur = 0
		}
		start := cur - maxVisible/2
		if start < 0 {
			start = 0
		}
		end := start + maxVisible
		if end > len(m.playlistTracks) {
			end = len(m.playlistTracks)
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
		if start > 0 {
			sb.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render(
				fmt.Sprintf("  ↑ %d tracks above", start)) + "\n")
		}
		for i := start; i < end; i++ {
			t := m.playlistTracks[i]
			line := fmt.Sprintf("[%d] %s — %s", i+1, t.Title, t.Artist)
			if m.playlistDetailIndex == i+2 {
				sb.WriteString(highlightStyle.Render("> "+line) + "\n")
			} else {
				sb.WriteString(normalStyle.Render("  "+line) + "\n")
			}
		}
		if end < len(m.playlistTracks) {
			sb.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render(
				fmt.Sprintf("  ↓ %d tracks below", len(m.playlistTracks)-end)) + "\n")
		}
	}

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
		Padding(0, 1).Width(cardWidth).
		Render(fmt.Sprintf("🎵 %s (Esc to go back)\n\n", m.selectedPlaylist.Name) + sb.String())
	return box + "\n"
}

func (m model) renderSearchInput(cardWidth int) string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
		Padding(0, 1).Width(cardWidth).
		Render("🔍 SEARCH LIBRARY (Esc to go back)\n\n" + m.searchInput.View() +
			"\n\n" + helpStyle.Render("Enter to search"))
	return box + "\n"
}

func (m model) renderSearchResults(cardWidth int) string {
	var sb strings.Builder
	header := "🔍 SEARCH RESULTS (Esc to go back)"
	if m.loading {
		header = "🔍 SEARCHING... (Esc to go back)"
		sb.WriteString(helpStyle.Render("  Searching your library...") + "\n")
	} else if len(m.searchResults) == 0 {
		sb.WriteString(normalStyle.Render("  No tracks found.") + "\n")
	} else {
		header = fmt.Sprintf("🔍 %d RESULT(S)  (Esc to go back)", len(m.searchResults))
		const maxVisible = 14
		cur := m.searchIndex
		start := cur - maxVisible/2
		if start < 0 {
			start = 0
		}
		end := start + maxVisible
		if end > len(m.searchResults) {
			end = len(m.searchResults)
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
		if start > 0 {
			sb.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render(
				fmt.Sprintf("  \xe2\x86\x91 %d more above", start)) + "\n")
		}
		for i := start; i < end; i++ {
			t := m.searchResults[i]
			line := fmt.Sprintf("%s \xe2\x80\x94 %s  (%s)", t.Title, t.Artist, t.Album)
			if i == m.searchIndex {
				sb.WriteString(highlightStyle.Render("> "+line) + "\n")
			} else {
				sb.WriteString(normalStyle.Render("  "+line) + "\n")
			}
		}
		if end < len(m.searchResults) {
			sb.WriteString(lipgloss.NewStyle().Foreground(subtleColor).Render(
				fmt.Sprintf("  \xe2\x86\x93 %d more below", len(m.searchResults)-end)) + "\n")
		}
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
		Padding(0, 1).Width(cardWidth).Render(header + "\n\n" + sb.String())
	return box + "\n"
}

func (m model) renderLyrics(cardWidth int) string {
	var sb strings.Builder

	if m.lyricsLoading {
		sb.WriteString(helpStyle.Render("  Fetching lyrics...") + "\n")
	} else if m.lyricsErr != "" {
		errMsg := m.lyricsErr
		// Never show raw URLs in the UI
		if strings.Contains(errMsg, "://") {
			errMsg = "Could not reach the lyrics service"
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Render("  ✕  "+errMsg) + "\n")
		sb.WriteString(helpStyle.Render("  Try pressing Esc and re-opening Lyrics to retry.") + "\n")
	} else if len(m.lyricsLines) == 0 {
		sb.WriteString(normalStyle.Render("  No lyrics found for this track.") + "\n")
	} else {
		active := lyricslib.ActiveIndex(m.lyricsLines, m.currentTrack.Position)

		// Show a window of lines centred on the active one
		const halfWindow = 6
		start := active - halfWindow
		if start < 0 {
			start = 0
		}
		end := start + halfWindow*2 + 1
		if end > len(m.lyricsLines) {
			end = len(m.lyricsLines)
		}

		for i := start; i < end; i++ {
			text := m.lyricsLines[i].Text
			if text == "" {
				sb.WriteString("\n")
				continue
			}
			if i == active {
				sb.WriteString(highlightStyle.Render("  "+text) + "\n")
			} else {
				dim := lipgloss.NewStyle().Foreground(subtleColor).Padding(0, 1)
				sb.WriteString(dim.Render("  "+text) + "\n")
			}
		}
	}

	synced := lyricslib.IsSynced(m.lyricsLines)
	syncLabel := ""
	if synced {
		syncLabel = " ·  synced"
	}
	header := fmt.Sprintf("♩ LYRICS%s  (Esc to go back)", syncLabel)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
		Padding(0, 1).Width(cardWidth).Render(header + "\n\n" + sb.String())
	return box + "\n"
}

func (m model) renderQueue(cardWidth int) string {
	var sb strings.Builder
	if m.queueLoading {
		sb.WriteString(helpStyle.Render("  Loading queue...") + "\n")
	} else if len(m.queueTracks) == 0 {
		sb.WriteString(normalStyle.Render("  No upcoming tracks.") + "\n")
	} else {
		for i, t := range m.queueTracks {
			line := fmt.Sprintf("[%d]  %s — %s", i+1, t.Title, t.Artist)
			if i == m.queueIndex {
				sb.WriteString(highlightStyle.Render("> "+line) + "\n")
			} else {
				sb.WriteString(normalStyle.Render("  "+line) + "\n")
			}
		}
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
		Padding(0, 1).Width(cardWidth).Render("⏩ UP NEXT  (Enter to play  ·  Esc to go back)\n\n" + sb.String())
	return box + "\n"
}

func (m model) renderHistory(cardWidth int) string {
	var sb strings.Builder
	if m.historyLoading {
		sb.WriteString(helpStyle.Render("  Loading history & stats...") + "\n")
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
			Padding(0, 1).Width(cardWidth).Render("📊 HISTORY & STATS  (Esc to go back)\n\n" + sb.String())
		return box + "\n"
	}
	if m.historyError != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3B30")).Render("  ✕  "+m.historyError) + "\n")
		box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
			Padding(0, 1).Width(cardWidth).Render("📊 HISTORY & STATS  (Esc to go back)\n\n" + sb.String())
		return box + "\n"
	}

	sectionHeader := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	dim := lipgloss.NewStyle().Foreground(subtleColor)

	// Library stats section
	if m.historyLibraryStats != nil {
		stats := m.historyLibraryStats
		sb.WriteString(sectionHeader.Render("  LIBRARY OVERVIEW") + "\n")
		sb.WriteString(dim.Render("  ─────────────────────────────────────────") + "\n")
		sb.WriteString(fmt.Sprintf("  Total Tracks:     %d\n", stats.TotalTracks))
		sb.WriteString(fmt.Sprintf("  Total Playlists:  %d\n", stats.TotalPlaylists))
		sb.WriteString(fmt.Sprintf("  Total Plays:      %d\n", stats.TotalPlayCount))
		if stats.MostPlayedGenre != "" {
			sb.WriteString(fmt.Sprintf("  Top Genre:        %s\n", stats.MostPlayedGenre))
		}
		if stats.TopArtist != "" {
			sb.WriteString(fmt.Sprintf("  Top Artist:       %s\n", stats.TopArtist))
		}
		sb.WriteString("\n")
	}

	// Recently played section
	if len(m.historyRecentTracks) > 0 {
		sb.WriteString(sectionHeader.Render("  RECENTLY PLAYED") + "\n")
		sb.WriteString(dim.Render("  ─────────────────────────────────────────") + "\n")
		for i, t := range m.historyRecentTracks {
			if i >= 10 {
				break
			}
			sb.WriteString(fmt.Sprintf("  %s — %s\n", t.Title, t.Artist))
		}
		sb.WriteString("\n")
	}

	// Top tracks section
	if len(m.historyTopTracks) > 0 {
		sb.WriteString(sectionHeader.Render("  TOP TRACKS (by play count)") + "\n")
		sb.WriteString(dim.Render("  ─────────────────────────────────────────") + "\n")
		for i, t := range m.historyTopTracks {
			if i >= 10 {
				break
			}
			playLabel := fmt.Sprintf("%d plays", t.PlayCount)
			sb.WriteString(fmt.Sprintf("  %2d. %-30s  %-20s %s\n", i+1, t.Title, t.Artist, playLabel))
		}
	}

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).
		Padding(0, 1).Width(cardWidth).Render("📊 HISTORY & STATS  (Esc to go back)\n\n" + sb.String())
	return box + "\n"
}

// ── Wide title ────────────────────────────────────────────────────────────────

// toFullwidth converts a string to Unicode fullwidth characters.
// Each character becomes 2 terminal cells wide, making the title visually 2× larger.
// Lipgloss correctly measures fullwidth chars so wrapping works automatically.
func toFullwidth(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			sb.WriteRune(0xFF21 + (r - 'A')) // Ａ–Ｚ
		case r >= 'a' && r <= 'z':
			sb.WriteRune(0xFF21 + (r - 'a')) // convert to fullwidth uppercase
		case r >= '0' && r <= '9':
			sb.WriteRune(0xFF10 + (r - '0')) // ０–９
		case r == ' ':
			sb.WriteRune(0x3000) // ideographic space (2 cells)
		case r == '(':
			sb.WriteRune(0xFF08)
		case r == ')':
			sb.WriteRune(0xFF09)
		case r == '-':
			sb.WriteRune(0xFF0D)
		case r == '&':
			sb.WriteRune(0xFF06)
		case r == '\'':
			sb.WriteRune(0xFF07)
		case r == '!':
			sb.WriteRune(0xFF01)
		case r == '.':
			sb.WriteRune(0xFF0E)
		case r == ',':
			sb.WriteRune(0xFF0C)
		case r == '/':
			sb.WriteRune(0xFF0F)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// renderBigTitle returns the track title as fullwidth Unicode constrained to
// the metadata column. Long titles are capped at two visual lines.
func renderBigTitle(title string, width int) string {
	if width <= 0 {
		return ""
	}
	full := toFullwidth(title)
	lines := wrapDisplayWidth(full, width, 2)
	return strings.Join(lines, "\n")
}

func wrapDisplayWidth(s string, width, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return nil
	}

	var lines []string
	var line strings.Builder
	lineW := 0
	runes := []rune(s)
	for i, r := range runes {
		rw := lipgloss.Width(string(r))
		if lineW > 0 && lineW+rw > width {
			lines = append(lines, line.String())
			line.Reset()
			lineW = 0
			if len(lines) == maxLines {
				lines[maxLines-1] = trimDisplayWidth(lines[maxLines-1], width-1) + "…"
				return lines
			}
		}
		line.WriteRune(r)
		lineW += rw
		if i == len(runes)-1 {
			lines = append(lines, line.String())
		}
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func trimDisplayWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	var out strings.Builder
	used := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if used+rw > width {
			break
		}
		out.WriteRune(r)
		used += rw
	}
	return out.String()
}

// ── RunTUI ────────────────────────────────────────────────────────────────────

func RunTUI() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ── RunMini ───────────────────────────────────────────────────────────────────

// RunMini displays a compact single-line now-playing bar that updates every second.
// Ideal for a tmux status pane or a tiny terminal window.
func RunMini() error {
	// Hide cursor
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h\n")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			return nil
		case <-ticker.C:
			info, err := musicAPI.NowPlaying()
			var line string
			if err != nil {
				line = fmt.Sprintf("Error: %v", err)
			} else {
				line = miniLine(info)
			}
			// Overwrite current line (pad to 120 chars to clear leftovers, truncate if needed)
			if len(line) > 120 {
				line = line[:117] + "..."
			}
			fmt.Printf("\r%-120s", line)
		}
	}
}

func miniLine(info *music.TrackInfo) string {
	if info == nil || info.State == "stopped" {
		return "⏹  Music stopped"
	}

	stateIcon := "▶"
	if info.State == "paused" {
		stateIcon = "⏸"
	}

	// Progress bar (18 chars)
	bar := ""
	if info.Duration > 0 {
		filled := int(info.Position / info.Duration * 16)
		if filled > 16 {
			filled = 16
		}
		bar = strings.Repeat("━", filled) + "●" + strings.Repeat("─", 16-filled)
		bar = "  " + bar + "  "
	}

	pos := fmt.Sprintf("%02d:%02d", int(info.Position)/60, int(info.Position)%60)
	dur := fmt.Sprintf("%02d:%02d", int(info.Duration)/60, int(info.Duration)%60)

	loved := ""
	if info.Loved {
		loved = "  ♥"
	}

	return fmt.Sprintf("%s  %s — %s%s%s / %s  ♪ %d%%%s",
		stateIcon, info.Title, info.Artist, bar, pos, dur, info.Volume, loved)
}
