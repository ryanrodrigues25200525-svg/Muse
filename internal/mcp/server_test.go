package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/lyrics"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/music"
)

func TestAvailableToolsIncludesExpectedSchemas(t *testing.T) {
	tools := availableTools()
	names := map[string]object{}
	for _, tool := range tools {
		name, ok := tool["name"].(string)
		if !ok || name == "" {
			t.Fatalf("tool has invalid name: %#v", tool)
		}
		names[name] = tool
	}

	for _, name := range []string{
		"play", "pause", "toggle", "next", "previous", "stop",
		"now_playing", "set_volume", "search", "play_song", "play_playlist",
		"lyrics", "queue", "stats", "love", "shuffle", "repeat", "seek",
		"play_playlist_shuffled", "music_context",
		"get_playlist_tracks", "list_playlists", "jump_to_track", "create_playlist",
		"add_to_playlist", "remove_from_playlist", "get_queue", "shuffle_mode",
		"repeat_mode", "get_volume", "fade_out", "get_song_info", "get_lyrics",
		"get_album", "rate_song", "love_song", "dislike_song", "get_recently_played",
		"top_tracks", "get_recommendations", "radio_mode", "add_to_queue",
		"add_to_library", "import_playlist", "search_filtered",
	} {
		if _, ok := names[name]; !ok {
			t.Fatalf("missing tool %q in %#v", name, names)
		}
	}

	setVolume := names["set_volume"]
	schema, ok := setVolume["inputSchema"].(object)
	if !ok {
		t.Fatalf("set_volume inputSchema = %#v", setVolume["inputSchema"])
	}
	required, ok := schema["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "volume" {
		t.Fatalf("set_volume required = %#v, want [volume]", schema["required"])
	}
}

func TestExecuteToolValidatesArgumentsBeforeCallingMusic(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{name: "set_volume", args: nil},
		{name: "search", args: nil},
		{name: "play_song", args: nil},
		{name: "play_playlist", args: nil},
		{name: "play_playlist_shuffled", args: nil},
		{name: "seek", args: nil},
		{name: "get_playlist_tracks", args: nil},
		{name: "jump_to_track", args: nil},
		{name: "create_playlist", args: nil},
		{name: "add_to_playlist", args: nil},
		{name: "remove_from_playlist", args: nil},
		{name: "repeat_mode", args: nil},
		{name: "rate_song", args: nil},
		{name: "unknown", args: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isErr := executeTool(tt.name, tt.args)
			if !isErr {
				t.Fatalf("executeTool(%q) isErr = false, result %q", tt.name, result)
			}
			if result == "" {
				t.Fatalf("executeTool(%q) returned empty error text", tt.name)
			}
		})
	}
}

func TestExecuteToolPlaylistToolsReturnStructuredJSON(t *testing.T) {
	withStubbedMusic(t)

	result, isErr := executeTool("list_playlists", nil)
	if isErr {
		t.Fatalf("executeTool(list_playlists) isErr = true, result = %q", result)
	}
	if !strings.Contains(result, "mine") || !strings.Contains(result, "others") {
		t.Fatalf("list_playlists result = %q", result)
	}

	result, isErr = executeTool("get_playlist_tracks", map[string]interface{}{"name": "Gym"})
	if isErr {
		t.Fatalf("executeTool(get_playlist_tracks) isErr = true, result = %q", result)
	}
	var tracks []music.TrackInfo
	if err := json.Unmarshal([]byte(result), &tracks); err != nil {
		t.Fatalf("playlist tracks result is not JSON: %v\n%s", err, result)
	}
	if len(tracks) != 1 || tracks[0].Title != "Song" {
		t.Fatalf("tracks = %#v", tracks)
	}
}

func TestExecuteToolReturnsStructuredSearchJSON(t *testing.T) {
	withStubbedMusic(t)

	result, isErr := executeTool("search", map[string]interface{}{"query": "song"})
	if isErr {
		t.Fatalf("executeTool(search) isErr = true, result = %q", result)
	}

	var tracks []music.TrackInfo
	if err := json.Unmarshal([]byte(result), &tracks); err != nil {
		t.Fatalf("search result is not JSON: %v\n%s", err, result)
	}
	if len(tracks) != 1 || tracks[0].Title != "Song" {
		t.Fatalf("tracks = %#v", tracks)
	}
}

func TestExecuteToolSearchFilteredReturnsJSON(t *testing.T) {
	withStubbedMusic(t)

	result, isErr := executeTool("search_filtered", map[string]interface{}{"artist": "Artist"})
	if isErr {
		t.Fatalf("executeTool(search_filtered) isErr = true, result = %q", result)
	}

	var tracks []music.TrackInfo
	if err := json.Unmarshal([]byte(result), &tracks); err != nil {
		t.Fatalf("search_filtered result is not JSON: %v\n%s", err, result)
	}
	if len(tracks) != 1 || tracks[0].Title != "Song" {
		t.Fatalf("tracks = %#v", tracks)
	}
}

func TestExecuteToolMusicContextIncludesAvailableData(t *testing.T) {
	withStubbedMusic(t)

	result, isErr := executeTool("music_context", nil)
	if isErr {
		t.Fatalf("executeTool(music_context) isErr = true, result = %q", result)
	}
	for _, want := range []string{"now_playing", "stats", "queue", "lyrics"} {
		if !strings.Contains(result, want) {
			t.Fatalf("music_context result = %q, want %q", result, want)
		}
	}
}

func withStubbedMusic(t *testing.T) {
	t.Helper()
	original := musicAPI
	musicAPI = musicActions{
		Play:   func() error { return nil },
		Pause:  func() error { return nil },
		Toggle: func() error { return nil },
		Next:   func() error { return nil },
		Prev:   func() error { return nil },
		Stop:   func() error { return nil },
		SetVolume: func(int) error {
			return nil
		},
		NowPlaying: func() (*music.TrackInfo, error) {
			return &music.TrackInfo{
				Title:    "Song",
				Artist:   "Artist",
				Album:    "Album",
				State:    "playing",
				Duration: 180,
			}, nil
		},
		Search: func(string) ([]music.TrackInfo, error) {
			return []music.TrackInfo{{Title: "Song", Artist: "Artist", Album: "Album"}}, nil
		},
		SearchFiltered: func(string, string, string, string, int, *bool, int, int) ([]music.TrackInfo, error) {
			return []music.TrackInfo{{Title: "Song", Artist: "Artist", Album: "Album"}}, nil
		},
		PlayTrackByName:      func(string) error { return nil },
		PlayPlaylist:         func(string) error { return nil },
		PlayPlaylistShuffled: func(string) error { return nil },
		GetCategorizedPlaylistInfos: func() ([]music.PlaylistInfo, []music.PlaylistInfo, error) {
			return []music.PlaylistInfo{{Name: "Gym", TrackCount: 1}}, []music.PlaylistInfo{{Name: "Replay", TrackCount: 2}}, nil
		},
		GetPlaylistTracks: func(string) ([]music.TrackInfo, error) {
			return []music.TrackInfo{{Title: "Song", Artist: "Artist", Album: "Album"}}, nil
		},
		PlayPlaylistTrackAtIndex: func(string, int) error { return nil },
		CreatePlaylist:           func(string) error { return nil },
		AddTrackToPlaylist:       func(string, string) error { return nil },
		RemoveTrackFromPlaylist:  func(string, string, int) error { return nil },
		GetQueue: func() ([]music.TrackInfo, error) {
			return []music.TrackInfo{{Title: "Next", Artist: "Artist"}}, nil
		},
		GetTrackStats: func() (*music.TrackStats, error) {
			return &music.TrackStats{Title: "Song", Artist: "Artist", PlayCount: 7, Rating: 80}, nil
		},
		GetVolume:                func() (int, error) { return 50, nil },
		ToggleLove:               func() (bool, error) { return true, nil },
		SetLove:                  func(bool) (bool, error) { return true, nil },
		DislikeCurrentTrack:      func() error { return nil },
		RateCurrentTrack:         func(int) error { return nil },
		ToggleShuffle:            func() (bool, error) { return true, nil },
		SetShuffleMode:           func(bool) error { return nil },
		CycleRepeat:              func() (string, error) { return "all", nil },
		SetRepeatMode:            func(string) error { return nil },
		Seek:                     func(float64) error { return nil },
		SetPlayerPosition:        func(float64) error { return nil },
		FadeOut:                  func(float64) error { return nil },
		PlayAlbumByName:          func(string) error { return nil },
		AddCurrentTrackToLibrary: func() error { return nil },
		GetRecentlyPlayed:        func(int) ([]music.TrackInfo, error) { return []music.TrackInfo{{Title: "Recent"}}, nil },
		GetTopTracks:             func(int) ([]music.TrackStats, error) { return []music.TrackStats{{Title: "Top", PlayCount: 9}}, nil },
		FetchLyrics: func(string, string, string, float64) ([]lyrics.Line, error) {
			return []lyrics.Line{{Time: 1.2, Text: "Line"}}, nil
		},
	}
	t.Cleanup(func() { musicAPI = original })
}
