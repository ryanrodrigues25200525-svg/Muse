package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/music"
)

func press(m model, key string) (model, tea.Cmd) {
	next, cmd := m.Update(testKey(key))
	return next.(model), cmd
}

func testKey(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

func withStubbedMusic(t *testing.T) *[]string {
	t.Helper()

	original := musicAPI
	calls := []string{}
	musicAPI = musicActions{
		NowPlaying: func() (*music.TrackInfo, error) {
			calls = append(calls, "now")
			return &music.TrackInfo{State: "playing", Volume: 50}, nil
		},
		GetCategorizedPlaylistInfos: func() ([]music.PlaylistInfo, []music.PlaylistInfo, error) {
			calls = append(calls, "playlists")
			return []music.PlaylistInfo{{Name: "Mine", PersistentID: "mine", TrackCount: 2}}, []music.PlaylistInfo{{Name: "Other", PersistentID: "other", TrackCount: 1}}, nil
		},
		GetPlaylistTracksByPersistentID: func(pid string) ([]music.TrackInfo, error) {
			calls = append(calls, "playlist-tracks:"+pid)
			return []music.TrackInfo{{Title: "One"}, {Title: "Two"}}, nil
		},
		Search: func(query string) ([]music.TrackInfo, error) {
			calls = append(calls, "search:"+query)
			return []music.TrackInfo{{Title: "Result"}}, nil
		},
		GetQueue: func() ([]music.TrackInfo, error) {
			calls = append(calls, "queue")
			return []music.TrackInfo{{Title: "Queued"}}, nil
		},
		GetArtworkPath: func() (string, error) {
			calls = append(calls, "art")
			return "", nil
		},
		RunAppleScript: func(script string) (string, error) {
			calls = append(calls, "script")
			return "", nil
		},
		Toggle: func() error {
			calls = append(calls, "toggle")
			return nil
		},
		Next: func() error {
			calls = append(calls, "next")
			return nil
		},
		Prev: func() error {
			calls = append(calls, "prev")
			return nil
		},
		CycleRepeat: func() (string, error) {
			calls = append(calls, "repeat")
			return "all", nil
		},
		ToggleShuffle: func() (bool, error) {
			calls = append(calls, "shuffle")
			return true, nil
		},
		ToggleLove: func() (bool, error) {
			calls = append(calls, "love")
			return true, nil
		},
		Seek: func(delta float64) error {
			calls = append(calls, "seek")
			return nil
		},
		SetVolume: func(volume int) error {
			calls = append(calls, "volume")
			return nil
		},
		Play: func() error {
			calls = append(calls, "play")
			return nil
		},
		Pause: func() error {
			calls = append(calls, "pause")
			return nil
		},
		PlayPlaylistByPersistentID: func(pid string) error {
			calls = append(calls, "playlist:"+pid)
			return nil
		},
		PlayPlaylistShuffledByPersistentID: func(pid string) error {
			calls = append(calls, "playlist-shuffle:"+pid)
			return nil
		},
		PlayPlaylistTrackAtIndexByPersistentID: func(pid string, index int) error {
			calls = append(calls, "playlist-track")
			return nil
		},
		PlayTrackByName: func(name string) error {
			calls = append(calls, "play-track:"+name)
			return nil
		},
		GetRecentlyPlayed: func(limit int) ([]music.TrackInfo, error) {
			calls = append(calls, "recent")
			return []music.TrackInfo{{Title: "Recent1", Artist: "Artist1"}, {Title: "Recent2", Artist: "Artist2"}}, nil
		},
		GetTopTracks: func(limit int) ([]music.TrackStats, error) {
			calls = append(calls, "top")
			return []music.TrackStats{{Title: "Top1", Artist: "A1", PlayCount: 10}, {Title: "Top2", Artist: "A2", PlayCount: 5}}, nil
		},
		GetLibraryStats: func() (*music.LibraryStats, error) {
			calls = append(calls, "libstats")
			return &music.LibraryStats{TotalTracks: 100, TotalPlaylists: 10, MostPlayedGenre: "Rock", TopArtist: "Artist1", TotalPlayCount: 500}, nil
		},
	}
	t.Cleanup(func() { musicAPI = original })
	return &calls
}

func assertCalls(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("calls = %#v, want %#v", got, want)
		}
	}
}

func TestMiniLineStopped(t *testing.T) {
	got := miniLine(&music.TrackInfo{State: "stopped"})
	if !strings.Contains(got, "Music stopped") {
		t.Fatalf("miniLine() = %q, want stopped message", got)
	}
}

func TestMiniLinePlayingIncludesCoreTrackInfo(t *testing.T) {
	got := miniLine(&music.TrackInfo{
		Title:    "Song",
		Artist:   "Artist",
		State:    "playing",
		Position: 65,
		Duration: 185,
		Volume:   72,
		Loved:    true,
	})

	for _, want := range []string{"Song", "Artist", "01:05", "03:05", "72%", "♥"} {
		if !strings.Contains(got, want) {
			t.Fatalf("miniLine() = %q, want it to contain %q", got, want)
		}
	}
}

func TestToFullwidth(t *testing.T) {
	got := toFullwidth("Abc 12!")
	want := "ＡＢＣ　１２！"
	if got != want {
		t.Fatalf("toFullwidth() = %q, want %q", got, want)
	}
}

func TestRenderBigTitleWrapsLongTitles(t *testing.T) {
	got := renderBigTitle("Dunno (feat. Dutchavelli & Stormzy)", 18)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("renderBigTitle() line count = %d, want 2; output %q", len(lines), got)
	}
	if !strings.HasSuffix(lines[1], "…") {
		t.Fatalf("renderBigTitle() final line = %q, want ellipsis", lines[1])
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 18 {
			t.Fatalf("line %q width = %d, want <= 18", line, width)
		}
	}
}

func TestInitialModelDefaultsToMainMenu(t *testing.T) {
	got := initialModel()
	if got.state != stateMainMenu {
		t.Fatalf("initialModel().state = %v, want %v", got.state, stateMainMenu)
	}
	if got.currentTrack == nil || got.currentTrack.State != "stopped" {
		t.Fatalf("initialModel().currentTrack = %#v, want stopped track", got.currentTrack)
	}
}

func TestRenderNowPlayingStoppedGivesActionableState(t *testing.T) {
	m := initialModel()
	got := m.renderNowPlaying(80)
	if !strings.Contains(got, "Open Music.app") {
		t.Fatalf("renderNowPlaying stopped = %q, want actionable Music.app message", got)
	}
}

func TestFriendlyErrorExplainsAutomationPermission(t *testing.T) {
	got := friendlyError("not authorized to send Apple events to Music")
	if !strings.Contains(got, "Automation access") {
		t.Fatalf("friendlyError() = %q, want Automation access guidance", got)
	}
}

func TestGlobalPlaybackKeybindsCallExpectedActions(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()
	m.currentTrack = &music.TrackInfo{State: "playing", Volume: 50}

	for _, key := range []string{"space", "n", "p", "r", "s", "l", "[", "]", "+", "-"} {
		m, _ = press(m, key)
	}

	assertCalls(t, *calls,
		"toggle",
		"next",
		"prev",
		"repeat",
		"shuffle",
		"love",
		"seek",
		"seek",
		"volume",
		"volume",
	)
}

func TestGlobalPlaybackKeybindsIgnoredInSearchInput(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()
	m.state = stateSearchInput
	m.currentTrack = &music.TrackInfo{State: "playing", Volume: 50}

	for _, key := range []string{"space", "n", "p", "r", "s", "l", "[", "]", "+", "-"} {
		m, _ = press(m, key)
	}

	assertCalls(t, *calls)
}

func TestControlErrorsAreShown(t *testing.T) {
	original := musicAPI
	musicAPI = musicActions{
		Toggle: func() error { return errors.New("toggle failed") },
		NowPlaying: func() (*music.TrackInfo, error) {
			return &music.TrackInfo{State: "playing", Volume: 50}, nil
		},
	}
	t.Cleanup(func() { musicAPI = original })

	m := initialModel()
	m.currentTrack = &music.TrackInfo{State: "playing", Volume: 50}

	m, _ = press(m, "space")
	if m.errorMsg != "toggle failed" {
		t.Fatalf("errorMsg = %q, want toggle failed", m.errorMsg)
	}
}

func TestDiscoToggleKey(t *testing.T) {
	withStubbedMusic(t)
	m := initialModel()

	m, _ = press(m, "d")
	if !m.discoMode {
		t.Fatal("disco mode should be enabled after pressing d")
	}

	m.discoPhase = 4
	m, _ = press(m, "d")
	if m.discoMode {
		t.Fatal("disco mode should be disabled after pressing d twice")
	}
	if m.discoPhase != 0 {
		t.Fatalf("discoPhase = %d, want 0 after disabling", m.discoPhase)
	}
}

func TestMainMenuNavigationAndEnterRoutes(t *testing.T) {
	withStubbedMusic(t)
	m := initialModel()

	m, _ = press(m, "down")
	if m.activeRow != 1 {
		t.Fatalf("activeRow = %d, want 1", m.activeRow)
	}
	m, _ = press(m, "right")
	if m.activeColumn != 1 {
		t.Fatalf("activeColumn = %d, want 1", m.activeColumn)
	}
	m, _ = press(m, "left")
	if m.activeColumn != 0 {
		t.Fatalf("activeColumn = %d, want 0", m.activeColumn)
	}
	m, _ = press(m, "up")
	if m.activeRow != 0 {
		t.Fatalf("activeRow = %d, want 0", m.activeRow)
	}

	m, _ = press(m, "enter")
	if m.state != statePlaylists {
		t.Fatalf("state = %v, want statePlaylists", m.state)
	}
}

func TestMainMenuControlButtonsCallExpectedActions(t *testing.T) {
	tests := []struct {
		name string
		row  int
		want string
	}{
		{name: "play", row: 0, want: "play"},
		{name: "pause", row: 1, want: "pause"},
		{name: "next", row: 2, want: "next"},
		{name: "previous", row: 3, want: "prev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := withStubbedMusic(t)
			m := initialModel()
			m.activeColumn = 1
			m.activeRow = tt.row

			m, _ = press(m, "enter")
			_ = m

			assertCalls(t, *calls, tt.want)
		})
	}
}

func TestMainMenuNavigationButtonsEnterStates(t *testing.T) {
	tests := []struct {
		name      string
		row       int
		wantState menuState
	}{
		{name: "playlists", row: 0, wantState: statePlaylists},
		{name: "search", row: 1, wantState: stateSearchInput},
		{name: "lyrics", row: 2, wantState: stateLyrics},
		{name: "queue", row: 3, wantState: stateQueue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withStubbedMusic(t)
			m := initialModel()
			m.currentTrack = &music.TrackInfo{Title: "Song", Artist: "Artist", State: "playing"}
			m.activeColumn = 0
			m.activeRow = tt.row

			m, _ = press(m, "enter")
			if m.state != tt.wantState {
				t.Fatalf("state = %v, want %v", m.state, tt.wantState)
			}
		})
	}
}

func TestEscapeReturnsToExpectedStates(t *testing.T) {
	withStubbedMusic(t)

	m := initialModel()
	m.state = statePlaylistDetail
	m, _ = press(m, "esc")
	if m.state != statePlaylists {
		t.Fatalf("state = %v, want statePlaylists", m.state)
	}

	for _, state := range []menuState{statePlaylists, stateSearchInput, stateSearchResults, stateLyrics, stateQueue, stateHistory} {
		m = initialModel()
		m.state = state
		m, _ = press(m, "esc")
		if m.state != stateMainMenu {
			t.Fatalf("state from %v = %v, want stateMainMenu", state, m.state)
		}
	}
}

func TestPlaylistListNavigationAndSelection(t *testing.T) {
	withStubbedMusic(t)
	m := initialModel()
	m.state = statePlaylists
	m.myPlaylists = []music.PlaylistInfo{{Name: "One", PersistentID: "one"}}
	m.otherPlaylists = []music.PlaylistInfo{{Name: "Two", PersistentID: "two"}}

	m, _ = press(m, "down")
	if m.playlistIndex != 1 {
		t.Fatalf("playlistIndex = %d, want 1", m.playlistIndex)
	}
	m, _ = press(m, "enter")
	if m.state != statePlaylistDetail {
		t.Fatalf("state = %v, want statePlaylistDetail", m.state)
	}
	if m.selectedPlaylist.PersistentID != "two" {
		t.Fatalf("selected playlist = %#v, want persistent ID two", m.selectedPlaylist)
	}
}

func TestPlaylistDetailDownStopsAtLastSelectableItem(t *testing.T) {
	withStubbedMusic(t)
	m := initialModel()
	m.state = statePlaylistDetail
	m.playlistTracks = []music.TrackInfo{{Title: "One"}, {Title: "Two"}}

	for i := 0; i < 10; i++ {
		m, _ = press(m, "down")
	}

	if m.playlistDetailIndex != 3 {
		t.Fatalf("playlistDetailIndex = %d, want 3", m.playlistDetailIndex)
	}
}

func TestPlaylistDetailButtonsCallExpectedActions(t *testing.T) {
	tests := []struct {
		name  string
		index int
		want  string
	}{
		{name: "play normally", index: 0, want: "playlist:pid"},
		{name: "play shuffled", index: 1, want: "playlist-shuffle:pid"},
		{name: "play track", index: 2, want: "playlist-track"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := withStubbedMusic(t)
			m := initialModel()
			m.state = statePlaylistDetail
			m.selectedPlaylist = music.PlaylistInfo{Name: "Playlist", PersistentID: "pid"}
			m.playlistTracks = []music.TrackInfo{{Title: "One"}, {Title: "Two"}}
			m.playlistDetailIndex = tt.index

			m, _ = press(m, "enter")

			assertCalls(t, *calls, tt.want)
			if m.state != stateMainMenu {
				t.Fatalf("state = %v, want stateMainMenu", m.state)
			}
			if tt.index == 2 && !m.hasPlaylistTrackContext() {
				t.Fatal("playlist track playback should set playlist track context")
			}
		})
	}
}

func TestFetchCommandsReturnExpectedMessages(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()

	msg := m.fetchPlaylistsCmd()().(playlistsMsg)
	if len(msg.mine) != 1 || msg.mine[0].PersistentID != "mine" {
		t.Fatalf("fetchPlaylistsCmd() = %#v", msg)
	}

	tracksMsg := m.fetchPlaylistTracksCmd(music.PlaylistInfo{PersistentID: "pid"})().(playlistTracksMsg)
	if len(tracksMsg.tracks) != 2 {
		t.Fatalf("fetchPlaylistTracksCmd() tracks = %#v, want 2 tracks", tracksMsg.tracks)
	}

	searchMsg := m.executeSearchCmd("abc")().(searchMsg)
	if len(searchMsg.tracks) != 1 || searchMsg.tracks[0].Title != "Result" {
		t.Fatalf("executeSearchCmd() = %#v", searchMsg)
	}

	queueMsg := fetchQueueCmd()().(queueMsg)
	if len(queueMsg.tracks) != 1 || queueMsg.tracks[0].Title != "Queued" {
		t.Fatalf("fetchQueueCmd() = %#v", queueMsg)
	}

	assertCalls(t, *calls, "playlists", "playlist-tracks:pid", "search:abc", "queue")
}

func TestSearchResultsEnterPlaysSelectedTrackAndClearsPlaylistContext(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()
	m.state = stateSearchResults
	m.searchResults = []music.TrackInfo{{Title: "One"}, {Title: "Two"}}
	m.searchIndex = 1
	m.activePlaylistID = "pid"
	m.activePlaylistTracks = []music.TrackInfo{{Title: "Old"}}

	m, _ = press(m, "enter")

	assertCalls(t, *calls, "play-track:Two")
	if m.hasPlaylistTrackContext() {
		t.Fatal("search playback should clear playlist context")
	}
}

func TestQueueEnterPlaysSelectedTrackAndClearsPlaylistContext(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()
	m.state = stateQueue
	m.queueTracks = []music.TrackInfo{{Title: "One"}, {Title: "Two"}}
	m.queueIndex = 1
	m.activePlaylistID = "pid"
	m.activePlaylistTracks = []music.TrackInfo{{Title: "Old"}}

	m, _ = press(m, "enter")

	assertCalls(t, *calls, "play-track:Two")
	if m.hasPlaylistTrackContext() {
		t.Fatal("queue playback should clear playlist context")
	}
}

func TestPlaylistContextNextPreviousButtons(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()
	m.activePlaylistID = "pid"
	m.activePlaylistTracks = []music.TrackInfo{{Title: "One"}, {Title: "Two"}, {Title: "Three"}}
	m.activePlaylistIndex = 1

	m, _ = press(m, "n")
	if m.activePlaylistIndex != 2 {
		t.Fatalf("activePlaylistIndex = %d, want 2", m.activePlaylistIndex)
	}
	m, _ = press(m, "p")
	if m.activePlaylistIndex != 1 {
		t.Fatalf("activePlaylistIndex = %d, want 1", m.activePlaylistIndex)
	}

	assertCalls(t, *calls, "playlist-track", "playlist-track")
}

func TestPlaylistContextNextPreviousWrapAtBoundaries(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()
	m.activePlaylistID = "pid"
	m.activePlaylistTracks = []music.TrackInfo{{Title: "One"}, {Title: "Two"}, {Title: "Three"}}
	m.activePlaylistIndex = 2

	m, _ = press(m, "n")
	if m.activePlaylistIndex != 0 {
		t.Fatalf("activePlaylistIndex after next = %d, want 0", m.activePlaylistIndex)
	}

	m, _ = press(m, "p")
	if m.activePlaylistIndex != 2 {
		t.Fatalf("activePlaylistIndex after previous = %d, want 2", m.activePlaylistIndex)
	}

	assertCalls(t, *calls, "playlist-track", "playlist-track")
}

func TestChangingPlaylistTrackReplacesNextContext(t *testing.T) {
	calls := withStubbedMusic(t)
	m := initialModel()

	m.state = statePlaylistDetail
	m.selectedPlaylist = music.PlaylistInfo{Name: "First", PersistentID: "first"}
	m.playlistTracks = []music.TrackInfo{{Title: "A"}, {Title: "B"}}
	m.playlistDetailIndex = 2
	m, _ = press(m, "enter")

	m.state = statePlaylistDetail
	m.selectedPlaylist = music.PlaylistInfo{Name: "Second", PersistentID: "second"}
	m.playlistTracks = []music.TrackInfo{{Title: "C"}, {Title: "D"}, {Title: "E"}}
	m.playlistDetailIndex = 3
	m, _ = press(m, "enter")

	if m.activePlaylistID != "second" {
		t.Fatalf("activePlaylistID = %q, want second", m.activePlaylistID)
	}
	if m.activePlaylistIndex != 1 {
		t.Fatalf("activePlaylistIndex = %d, want 1", m.activePlaylistIndex)
	}

	m, _ = press(m, "n")
	if m.activePlaylistIndex != 2 {
		t.Fatalf("activePlaylistIndex after next = %d, want 2", m.activePlaylistIndex)
	}

	assertCalls(t, *calls, "playlist-track", "playlist-track", "playlist-track")
}

func TestDiscoActiveRequiresModeAndPlayingTrack(t *testing.T) {
	tests := []struct {
		name  string
		model model
		want  bool
	}{
		{name: "disabled", model: model{currentTrack: &music.TrackInfo{State: "playing"}}, want: false},
		{name: "paused", model: model{discoMode: true, currentTrack: &music.TrackInfo{State: "paused"}}, want: false},
		{name: "playing", model: model{discoMode: true, currentTrack: &music.TrackInfo{State: "playing"}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.discoActive(); got != tt.want {
				t.Fatalf("discoActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrackChangeSyncsPlaylistIndex(t *testing.T) {
	withStubbedMusic(t)
	m := initialModel()
	m.activePlaylistID = "pid"
	m.activePlaylistTracks = []music.TrackInfo{
		{Title: "Song A", Artist: "Artist A", Album: "Album A"},
		{Title: "Song B", Artist: "Artist B", Album: "Album B"},
		{Title: "Song C", Artist: "Artist C", Album: "Album C"},
	}
	m.activePlaylistIndex = 0
	m.lastTrackTitle = "Song A"

	// 1. If a track change occurs naturally to a song in the playlist, index must update
	updated, _ := m.Update(statusMsg{
		info: &music.TrackInfo{
			Title:  "Song C",
			Artist: "Artist C",
			Album:  "Album C",
			State:  "playing",
		},
	})
	m2 := updated.(model)
	if m2.activePlaylistIndex != 2 {
		t.Fatalf("activePlaylistIndex = %d, want 2", m2.activePlaylistIndex)
	}
	if !m2.hasPlaylistTrackContext() {
		t.Fatal("should preserve playlist context")
	}

	// 2. If a track change occurs to a song not in the playlist, context must clear
	updated2, _ := m2.Update(statusMsg{
		info: &music.TrackInfo{
			Title:  "Song X",
			Artist: "Artist X",
			Album:  "Album X",
			State:  "playing",
		},
	})
	m3 := updated2.(model)
	if m3.hasPlaylistTrackContext() {
		t.Fatal("playlist context should be cleared when track is not in the playlist")
	}
}

func TestCurrentThemeCyclesWhenDiscoIsActive(t *testing.T) {
	got := model{
		discoMode:    true,
		discoPhase:   len(discoThemes) + 1,
		currentTrack: &music.TrackInfo{State: "playing"},
	}.currentTheme()

	want := discoThemes[1]
	if got != want {
		t.Fatalf("currentTheme() = %#v, want %#v", got, want)
	}
}

func TestHasPlaylistTrackContext(t *testing.T) {
	tracks := []music.TrackInfo{{Title: "One"}, {Title: "Two"}}

	tests := []struct {
		name  string
		model model
		want  bool
	}{
		{name: "missing playlist id", model: model{activePlaylistTracks: tracks, activePlaylistIndex: 0}, want: false},
		{name: "negative index", model: model{activePlaylistID: "pid", activePlaylistTracks: tracks, activePlaylistIndex: -1}, want: false},
		{name: "index too high", model: model{activePlaylistID: "pid", activePlaylistTracks: tracks, activePlaylistIndex: 2}, want: false},
		{name: "valid", model: model{activePlaylistID: "pid", activePlaylistTracks: tracks, activePlaylistIndex: 1}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.hasPlaylistTrackContext(); got != tt.want {
				t.Fatalf("hasPlaylistTrackContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCardWidthCapsWideTerminals(t *testing.T) {
	got := model{width: 160}.cardWidth()
	if got != 112 {
		t.Fatalf("cardWidth() = %d, want 112", got)
	}
}

func TestAlbumArtSizeScalesWithCardWidth(t *testing.T) {
	tests := []struct {
		name  string
		width int
		wantW int
		wantH int
	}{
		{name: "compact", width: 60, wantW: 22, wantH: 11},
		{name: "medium", width: 90, wantW: 28, wantH: 14},
		{name: "wide", width: 112, wantW: 34, wantH: 17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotW, gotH := albumArtSize(tt.width)
			if gotW != tt.wantW || gotH != tt.wantH {
				t.Fatalf("albumArtSize(%d) = %dx%d, want %dx%d", tt.width, gotW, gotH, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestHistoryMenuSelectEntersStateHistory(t *testing.T) {
	withStubbedMusic(t)
	m := initialModel()
	m.activeColumn = 0
	m.activeRow = 4 // "History & Stats"

	m, _ = press(m, "enter")
	if m.state != stateHistory {
		t.Fatalf("state = %v, want stateHistory", m.state)
	}
	if !m.historyLoading {
		t.Fatal("historyLoading should be true after entering history state")
	}
}

func TestFetchHistoryCmdReturnsExpectedMessages(t *testing.T) {
	withStubbedMusic(t)

	msg := fetchHistoryCmd()().(historyMsg)
	if len(msg.recentTracks) != 2 {
		t.Fatalf("fetchHistoryCmd().recentTracks = %d, want 2", len(msg.recentTracks))
	}
	if len(msg.topTracks) != 2 {
		t.Fatalf("fetchHistoryCmd().topTracks = %d, want 2", len(msg.topTracks))
	}
	if msg.libraryStats == nil {
		t.Fatal("fetchHistoryCmd().libraryStats is nil")
	}
	if msg.libraryStats.TotalTracks != 100 {
		t.Fatalf("fetchHistoryCmd().libraryStats.TotalTracks = %d, want 100", msg.libraryStats.TotalTracks)
	}
}

func TestRenderHistoryLoading(t *testing.T) {
	m := initialModel()
	m.state = stateHistory
	m.historyLoading = true
	got := m.renderHistory(60)
	if !strings.Contains(got, "Loading history") {
		t.Fatalf("renderHistory(loading) = %q, want loading message", got)
	}
}

func TestRenderHistoryWithData(t *testing.T) {
	m := initialModel()
	m.state = stateHistory
	m.historyRecentTracks = []music.TrackInfo{{Title: "Song1", Artist: "Art1"}, {Title: "Song2", Artist: "Art2"}}
	m.historyTopTracks = []music.TrackStats{{Title: "Top1", Artist: "A1", PlayCount: 10}, {Title: "Top2", Artist: "A2", PlayCount: 5}}
	m.historyLibraryStats = &music.LibraryStats{TotalTracks: 100, TotalPlaylists: 10, MostPlayedGenre: "Rock", TopArtist: "Artist1", TotalPlayCount: 500}
	got := m.renderHistory(80)
	if !strings.Contains(got, "LIBRARY OVERVIEW") {
		t.Fatalf("renderHistory should contain LIBRARY OVERVIEW, got %q", got)
	}
	if !strings.Contains(got, "Total Tracks:") {
		t.Fatalf("renderHistory should contain Total Tracks, got %q", got)
	}
	if !strings.Contains(got, "RECENTLY PLAYED") {
		t.Fatalf("renderHistory should contain RECENTLY PLAYED, got %q", got)
	}
	if !strings.Contains(got, "TOP TRACKS") {
		t.Fatalf("renderHistory should contain TOP TRACKS, got %q", got)
	}
	if !strings.Contains(got, "10 plays") {
		t.Fatalf("renderHistory should contain play counts, got %q", got)
	}
}
