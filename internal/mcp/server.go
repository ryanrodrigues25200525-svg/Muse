package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/lyrics"
	"github.com/ryanrodrigues25200525-svg/Apple-music-cli/internal/music"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]object `json:"capabilities"`
	ServerInfo      ServerInfo        `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type object map[string]interface{}

type musicActions struct {
	Play                        func() error
	Pause                       func() error
	Toggle                      func() error
	Next                        func() error
	Prev                        func() error
	Stop                        func() error
	NowPlaying                  func() (*music.TrackInfo, error)
	SetVolume                   func(int) error
	Search                      func(string) ([]music.TrackInfo, error)
	SearchFiltered              func(string, string, string, string, int, *bool, int, int) ([]music.TrackInfo, error)
	PlayTrackByName             func(string) error
	PlayPlaylist                func(string) error
	PlayPlaylistShuffled        func(string) error
	GetCategorizedPlaylistInfos func() ([]music.PlaylistInfo, []music.PlaylistInfo, error)
	GetPlaylistTracks           func(string) ([]music.TrackInfo, error)
	PlayPlaylistTrackAtIndex    func(string, int) error
	CreatePlaylist              func(string) error
	AddTrackToPlaylist          func(string, string) error
	RemoveTrackFromPlaylist     func(string, string, int) error
	GetQueue                    func() ([]music.TrackInfo, error)
	GetTrackStats               func() (*music.TrackStats, error)
	GetVolume                   func() (int, error)
	ToggleLove                  func() (bool, error)
	SetLove                     func(bool) (bool, error)
	DislikeCurrentTrack         func() error
	RateCurrentTrack            func(int) error
	ToggleShuffle               func() (bool, error)
	SetShuffleMode              func(bool) error
	CycleRepeat                 func() (string, error)
	SetRepeatMode               func(string) error
	Seek                        func(float64) error
	SetPlayerPosition           func(float64) error
	FadeOut                     func(float64) error
	FadeIn                      func(float64, int) error
	PlayAlbumByName             func(string) error
	AddCurrentTrackToLibrary    func() error
	GetRecentlyPlayed           func(int) ([]music.TrackInfo, error)
	GetTopTracks                func(int) ([]music.TrackStats, error)
	ExportPlaylist              func(string, string) (string, error)
	ImportPlaylist              func(string, string) error
	FetchLyrics                 func(string, string, string, float64) ([]lyrics.Line, error)
}

var (
	djMu       sync.Mutex
	responseMu sync.Mutex
)

var musicAPI = musicActions{
	Play:                        music.Play,
	Pause:                       music.Pause,
	Toggle:                      music.Toggle,
	Next:                        music.Next,
	Prev:                        music.Prev,
	Stop:                        music.Stop,
	NowPlaying:                  music.NowPlaying,
	SetVolume:                   music.SetVolume,
	Search:                      music.Search,
	SearchFiltered:              music.SearchFiltered,
	PlayTrackByName:             music.PlayTrackByName,
	PlayPlaylist:                music.PlayPlaylist,
	PlayPlaylistShuffled:        music.PlayPlaylistShuffled,
	GetCategorizedPlaylistInfos: music.GetCategorizedPlaylistInfos,
	GetPlaylistTracks:           music.GetPlaylistTracks,
	PlayPlaylistTrackAtIndex:    music.PlayPlaylistTrackAtIndex,
	CreatePlaylist:              music.CreatePlaylist,
	AddTrackToPlaylist:          music.AddTrackToPlaylist,
	RemoveTrackFromPlaylist:     music.RemoveTrackFromPlaylist,
	GetQueue:                    music.GetQueue,
	GetTrackStats:               music.GetTrackStats,
	GetVolume:                   music.GetVolume,
	ToggleLove:                  music.ToggleLove,
	SetLove:                     music.SetLove,
	DislikeCurrentTrack:         music.DislikeCurrentTrack,
	RateCurrentTrack:            music.RateCurrentTrack,
	ToggleShuffle:               music.ToggleShuffle,
	SetShuffleMode:              music.SetShuffleMode,
	CycleRepeat:                 music.CycleRepeat,
	SetRepeatMode:               music.SetRepeatMode,
	Seek:                        music.Seek,
	SetPlayerPosition:           music.SetPlayerPosition,
	FadeOut:                     music.FadeOut,
	FadeIn:                      music.FadeIn,
	PlayAlbumByName:             music.PlayAlbumByName,
	AddCurrentTrackToLibrary:    music.AddCurrentTrackToLibrary,
	GetRecentlyPlayed:           music.GetRecentlyPlayed,
	GetTopTracks:                music.GetTopTracks,
	ExportPlaylist:              music.ExportPlaylist,
	ImportPlaylist:              music.ImportPlaylist,
	FetchLyrics:                 lyrics.Fetch,
}

func startDJMonitor() {
	go func() {
		for {
			idle := music.DJQueueLen() == 0 || music.DjFading
			if idle {
				time.Sleep(2 * time.Second)
				continue
			}
			time.Sleep(500 * time.Millisecond)

			info, err := musicAPI.NowPlaying()
			if err != nil || info == nil {
				continue
			}

			if music.DjFading || music.DJQueueLen() == 0 {
				continue
			}

			var doTransition bool
			var fadeTime float64
			switch {
			case info.State == "stopped":
				doTransition = true
			case info.State == "playing" && info.Duration > 0:
				left := info.Duration - info.Position
				if left > 0 && left <= music.DJFadeSecs+1 {
					doTransition = true
					fadeTime = left
				}
			}

			if !doTransition {
				continue
			}

			next, ok := music.DJDequeueNext()
			if !ok {
				continue
			}

			music.DjFading = true
			origVol, err := musicAPI.GetVolume()
			if err != nil || origVol <= 0 {
				origVol = 80
			}
			if fadeTime > 0.3 {
				musicAPI.FadeOut(fadeTime)
			} else {
				musicAPI.SetVolume(0)
			}
			musicAPI.PlayTrackByName(next.Title)
			musicAPI.FadeIn(music.DJFadeSecs, origVol)

			music.DjFading = false
		}
	}()
}

func StartServer() {
	startDJMonitor()
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		handleRequest(&req)
	}
}

func sendResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	bz, err := json.Marshal(resp)
	if err == nil {
		os.Stdout.Write(bz)
		os.Stdout.Write([]byte("\n"))
	}
}

func sendError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	bz, err := json.Marshal(resp)
	if err == nil {
		os.Stdout.Write(bz)
		os.Stdout.Write([]byte("\n"))
	}
}

func handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		sendResponse(req.ID, InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]object{
				"tools": {},
			},
			ServerInfo: ServerInfo{
				Name:    "muse-mcp",
				Version: "0.1.0",
			},
		})

	case "notifications/initialized":
		// No-op

	case "tools/list":
		sendResponse(req.ID, object{"tools": availableTools()})

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.ID, -32602, "Invalid params", err.Error())
			return
		}

		result, isErr := executeTool(params.Name, params.Arguments)
		sendResponse(req.ID, object{
			"content": []object{
				{
					"type": "text",
					"text": result,
				},
			},
			"isError": isErr,
		})

	default:
		sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

func availableTools() []object {
	return []object{
		{
			"name":        "play",
			"description": "Resume playback or start playing music",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "pause",
			"description": "Pause music playback",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "toggle",
			"description": "Toggle play/pause state",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "next",
			"description": "Skip to the next song",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "previous",
			"description": "Skip to the previous song",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "stop",
			"description": "Stop music playback",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "now_playing",
			"description": "Retrieve information about the currently playing track",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "set_volume",
			"description": "Set the Music.app volume (0-100)",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"volume": object{
						"type":        "integer",
						"description": "Volume level between 0 and 100",
					},
				},
				"required": []string{"volume"},
			},
		},
		{
			"name":        "search",
			"description": "Search the user's music library for songs by title, artist, or album",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"query": object{
						"type":        "string",
						"description": "The search term (song title, artist, or album name)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "play_song",
			"description": "Search for and play a specific song by its name, artist, or album",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"query": object{
						"type":        "string",
						"description": "The name, artist, or album of the song to play",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "play_playlist",
			"description": "Play a playlist by name",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"name": object{
						"type":        "string",
						"description": "The exact name of the playlist to play",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "play_playlist_shuffled",
			"description": "Enable shuffle and play a playlist by name",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"name": object{
						"type":        "string",
						"description": "The exact name of the playlist to play",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "list_playlists",
			"description": "List user, library, Apple Music, and shared playlists",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "get_playlist_tracks",
			"description": "List tracks in a playlist by name",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"name": object{
						"type":        "string",
						"description": "The exact name of the playlist",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "jump_to_track",
			"description": "Play a 1-based track index within a playlist",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"playlist": object{
						"type":        "string",
						"description": "The exact playlist name",
					},
					"index": object{
						"type":        "integer",
						"description": "1-based track index in the playlist",
					},
				},
				"required": []string{"playlist", "index"},
			},
		},
		{
			"name":        "create_playlist",
			"description": "Create a new user playlist",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"name": object{
						"type":        "string",
						"description": "Playlist name",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "add_to_playlist",
			"description": "Add the first matching library song to an existing playlist",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"playlist": object{
						"type":        "string",
						"description": "The exact playlist name",
					},
					"query": object{
						"type":        "string",
						"description": "Song title, artist, or album to find in the library",
					},
				},
				"required": []string{"playlist", "query"},
			},
		},
		{
			"name":        "remove_from_playlist",
			"description": "Remove a song from a playlist by query or 1-based index",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"playlist": object{
						"type":        "string",
						"description": "The exact playlist name",
					},
					"query": object{
						"type":        "string",
						"description": "Optional song title, artist, or album match",
					},
					"index": object{
						"type":        "integer",
						"description": "Optional 1-based track index",
					},
				},
				"required": []string{"playlist"},
			},
		},
		{
			"name":        "lyrics",
			"description": "Fetch lyrics for the currently playing track",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "queue",
			"description": "Return upcoming tracks in the current playlist",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "get_queue",
			"description": "Return upcoming tracks in the current playlist",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "stats",
			"description": "Return play count, rating, loved status, and metadata for the current track",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "get_song_info",
			"description": "Return title, artist, album, duration, play count, rating, and loved status for the current track",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "get_lyrics",
			"description": "Fetch lyrics for the currently playing track",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "get_album",
			"description": "Play an album from the local library by album name",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"query": object{
						"type":        "string",
						"description": "Album name to find in the library",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "rate_song",
			"description": "Set current song star rating from 1 to 5",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"rating": object{
						"type":        "integer",
						"description": "Star rating from 1 to 5",
					},
				},
				"required": []string{"rating"},
			},
		},
		{
			"name":        "love",
			"description": "Toggle loved status for the current track",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "love_song",
			"description": "Mark the current track as loved",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "dislike_song",
			"description": "Mark the current track as disliked when Music.app supports it",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "shuffle",
			"description": "Toggle shuffle mode",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "shuffle_mode",
			"description": "Set shuffle on/off, or toggle when enabled is omitted",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"enabled": object{
						"type":        "boolean",
						"description": "Optional target shuffle state",
					},
				},
			},
		},
		{
			"name":        "repeat",
			"description": "Cycle repeat mode",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "repeat_mode",
			"description": "Set repeat mode to off, one, or all",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"mode": object{
						"type":        "string",
						"enum":        []string{"off", "one", "all"},
						"description": "Repeat mode",
					},
				},
				"required": []string{"mode"},
			},
		},
		{
			"name":        "seek",
			"description": "Seek by relative seconds or jump to an absolute timestamp",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"seconds": object{
						"type":        "number",
						"description": "Signed number of seconds to seek, for example 30 or -10",
					},
					"position": object{
						"type":        "number",
						"description": "Absolute timestamp in seconds from the start of the song",
					},
				},
			},
		},
		{
			"name":        "get_volume",
			"description": "Read Music.app volume",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "fade_out",
			"description": "Gradually fade Music.app volume to zero",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"seconds": object{
						"type":        "number",
						"description": "Fade duration in seconds. Defaults to 5.",
					},
				},
			},
		},
		{
			"name":        "get_recently_played",
			"description": "Return tracks from Music.app's Recently Played playlist when available",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"limit": object{
						"type":        "integer",
						"description": "Maximum tracks to return, default 25",
					},
				},
			},
		},
		{
			"name":        "top_tracks",
			"description": "Return the most-played tracks from the local library",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"limit": object{
						"type":        "integer",
						"description": "Maximum tracks to return, default 25",
					},
				},
			},
		},
		{
			"name":        "add_to_library",
			"description": "Save the current track to the local library when Music.app supports it",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "get_recommendations",
			"description": "Best-effort recommendations placeholder; Music.app does not expose For You recommendations to AppleScript",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "radio_mode",
			"description": "Best-effort radio placeholder; Music.app does not reliably expose station creation to AppleScript",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"query": object{
						"type":        "string",
						"description": "Optional song or artist seed",
					},
				},
			},
		},
		{
			"name":        "add_to_queue",
			"description": "Best-effort queue placeholder; Music.app does not expose Up Next mutation to AppleScript",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"query": object{
						"type":        "string",
						"description": "Song title, artist, or album",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "import_playlist",
			"description": "Best-effort import placeholder; playlist file imports are not supported by the local MCP server yet",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"path": object{
						"type":        "string",
						"description": "Playlist file path",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "music_context",
			"description": "Return now playing, stats, queue, and lyrics when available",
			"inputSchema": object{
				"type":       "object",
				"properties": object{},
			},
		},
		{
			"name":        "search_filtered",
			"description": "Search the user's music library with filters: artist, album, genre, year, loved status, minimum rating, and optional text query",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"query": object{
						"type":        "string",
						"description": "Optional text search (matches title, artist, album)",
					},
					"artist": object{
						"type":        "string",
						"description": "Filter by artist name",
					},
					"album": object{
						"type":        "string",
						"description": "Filter by album name",
					},
					"genre": object{
						"type":        "string",
						"description": "Filter by genre",
					},
					"year": object{
						"type":        "integer",
						"description": "Filter by release year",
					},
					"loved": object{
						"type":        "boolean",
						"description": "Filter by loved status",
					},
					"min_rating": object{
						"type":        "integer",
						"description": "Minimum rating (1-5 stars)",
					},
					"limit": object{
						"type":        "integer",
						"description": "Maximum results to return (1-100, default 50)",
					},
				},
			},
		},
		{
			"name":        "playlist_export",
			"description": "Export a Music.app playlist's tracks as M3U or JSON content",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"name": object{
						"type":        "string",
						"description": "The exact name of the playlist to export",
					},
					"format": object{
						"type":        "string",
						"description": "Export format: \"m3u\" (default) or \"json\"",
						"enum":        []string{"m3u", "json"},
					},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "playlist_import",
			"description": "Import tracks from an M3U or JSON file into a Music.app playlist (created if missing)",
			"inputSchema": object{
				"type": "object",
				"properties": object{
					"name": object{
						"type":        "string",
						"description": "Name of the target playlist",
					},
					"path_or_content": object{
						"type":        "string",
						"description": "File path to an M3U/JSON file, or the raw file content itself",
					},
					"format": object{
						"type":        "string",
						"description": "File format: \"m3u\" or \"json\". Auto-detected from file extension or content when omitted.",
						"enum":        []string{"m3u", "json"},
					},
				},
				"required": []string{"name", "path_or_content"},
			},
		},
	}
}

func executeTool(name string, args map[string]interface{}) (string, bool) {
	switch name {
	case "play":
		if err := musicAPI.Play(); err != nil {
			return fmt.Sprintf("Error playing: %v", err), true
		}
		return "Playback started/resumed.", false

	case "pause":
		if err := musicAPI.Pause(); err != nil {
			return fmt.Sprintf("Error pausing: %v", err), true
		}
		return "Playback paused.", false

	case "toggle":
		if err := musicAPI.Toggle(); err != nil {
			return fmt.Sprintf("Error toggling playback: %v", err), true
		}
		return "Playback toggled.", false

	case "next":
		if err := musicAPI.Next(); err != nil {
			return fmt.Sprintf("Error playing next track: %v", err), true
		}
		return "Skipped to next track.", false

	case "previous":
		if err := musicAPI.Prev(); err != nil {
			return fmt.Sprintf("Error playing previous track: %v", err), true
		}
		return "Skipped to previous track.", false

	case "stop":
		if err := musicAPI.Stop(); err != nil {
			return fmt.Sprintf("Error stopping: %v", err), true
		}
		return "Playback stopped.", false

	case "now_playing":
		info, err := musicAPI.NowPlaying()
		if err != nil {
			return fmt.Sprintf("Error fetching now playing: %v", err), true
		}
		return jsonText(info), false

	case "set_volume":
		// JSON numbers unmarshal as float64
		volFloat, ok := args["volume"].(float64)
		if !ok {
			return "volume argument is required and must be a number", true
		}
		vol := int(volFloat)
		if err := musicAPI.SetVolume(vol); err != nil {
			return fmt.Sprintf("Error setting volume: %v", err), true
		}
		return fmt.Sprintf("Volume set to %d%%.", vol), false

	case "search_filtered":
		query, _ := args["query"].(string)
		artist, _ := args["artist"].(string)
		album, _ := args["album"].(string)
		genre, _ := args["genre"].(string)
		var year int
		if y, ok := args["year"].(float64); ok {
			year = int(y)
		}
		var loved *bool
		if l, ok := args["loved"].(bool); ok {
			loved = &l
		}
		var minRating int
		if r, ok := args["min_rating"].(float64); ok {
			minRating = int(r)
		}
		var limit int
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		results, err := musicAPI.SearchFiltered(query, artist, album, genre, year, loved, minRating, limit)
		if err != nil {
			return fmt.Sprintf("Error searching library: %v", err), true
		}
		return jsonText(results), false

	case "search":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "query argument is required", true
		}
		results, err := musicAPI.Search(query)
		if err != nil {
			return fmt.Sprintf("Error searching library: %v", err), true
		}
		return jsonText(results), false

	case "play_song":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "query argument is required", true
		}
		if err := musicAPI.PlayTrackByName(query); err != nil {
			return fmt.Sprintf("Error playing track: %v", err), true
		}
		return fmt.Sprintf("Playing track matching: '%s'", query), false

	case "play_playlist":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return "name argument is required", true
		}
		if err := musicAPI.PlayPlaylist(name); err != nil {
			return fmt.Sprintf("Error playing playlist: %v", err), true
		}
		return fmt.Sprintf("Playing playlist: %s", name), false

	case "play_playlist_shuffled":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return "name argument is required", true
		}
		if err := musicAPI.PlayPlaylistShuffled(name); err != nil {
			return fmt.Sprintf("Error playing playlist shuffled: %v", err), true
		}
		return fmt.Sprintf("Playing playlist shuffled: %s", name), false

	case "list_playlists":
		mine, others, err := musicAPI.GetCategorizedPlaylistInfos()
		if err != nil {
			return fmt.Sprintf("Error listing playlists: %v", err), true
		}
		return jsonText(object{"mine": mine, "others": others}), false

	case "get_playlist_tracks":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return "name argument is required", true
		}
		tracks, err := musicAPI.GetPlaylistTracks(name)
		if err != nil {
			return fmt.Sprintf("Error fetching playlist tracks: %v", err), true
		}
		return jsonText(tracks), false

	case "jump_to_track":
		playlist, ok := args["playlist"].(string)
		if !ok || playlist == "" {
			return "playlist argument is required", true
		}
		index, ok := intArg(args, "index")
		if !ok || index <= 0 {
			return "index argument is required and must be 1 or greater", true
		}
		if err := musicAPI.PlayPlaylistTrackAtIndex(playlist, index); err != nil {
			return fmt.Sprintf("Error jumping to playlist track: %v", err), true
		}
		return jsonText(object{"playlist": playlist, "index": index}), false

	case "create_playlist":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return "name argument is required", true
		}
		if err := musicAPI.CreatePlaylist(name); err != nil {
			return fmt.Sprintf("Error creating playlist: %v", err), true
		}
		return fmt.Sprintf("Created playlist: %s", name), false

	case "add_to_playlist":
		playlist, ok := args["playlist"].(string)
		if !ok || playlist == "" {
			return "playlist argument is required", true
		}
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "query argument is required", true
		}
		if err := musicAPI.AddTrackToPlaylist(playlist, query); err != nil {
			return fmt.Sprintf("Error adding to playlist: %v", err), true
		}
		return fmt.Sprintf("Added track matching %q to playlist %q.", query, playlist), false

	case "remove_from_playlist":
		playlist, ok := args["playlist"].(string)
		if !ok || playlist == "" {
			return "playlist argument is required", true
		}
		query, _ := args["query"].(string)
		index, _ := intArg(args, "index")
		if query == "" && index <= 0 {
			return "query or 1-based index argument is required", true
		}
		if err := musicAPI.RemoveTrackFromPlaylist(playlist, query, index); err != nil {
			return fmt.Sprintf("Error removing from playlist: %v", err), true
		}
		return fmt.Sprintf("Removed track from playlist %q.", playlist), false

	case "lyrics", "get_lyrics":
		lines, err := lyricsForCurrentTrack()
		if err != nil {
			return fmt.Sprintf("Error fetching lyrics: %v", err), true
		}
		return jsonText(lines), false

	case "queue", "get_queue":
		tracks, err := musicAPI.GetQueue()
		if err != nil {
			return fmt.Sprintf("Error fetching queue: %v", err), true
		}
		return jsonText(tracks), false

	case "stats":
		stats, err := musicAPI.GetTrackStats()
		if err != nil {
			return fmt.Sprintf("Error fetching stats: %v", err), true
		}
		return jsonText(stats), false

	case "get_song_info":
		info, err := musicAPI.NowPlaying()
		if err != nil {
			return fmt.Sprintf("Error fetching now playing: %v", err), true
		}
		result := object{"now_playing": info}
		if stats, err := musicAPI.GetTrackStats(); err == nil {
			result["stats"] = stats
		} else {
			result["stats_error"] = err.Error()
		}
		return jsonText(result), false

	case "get_album":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "query argument is required", true
		}
		if err := musicAPI.PlayAlbumByName(query); err != nil {
			return fmt.Sprintf("Error playing album: %v", err), true
		}
		return fmt.Sprintf("Playing album matching: %s", query), false

	case "rate_song":
		rating, ok := intArg(args, "rating")
		if !ok {
			return "rating argument is required and must be 1-5", true
		}
		if err := musicAPI.RateCurrentTrack(rating); err != nil {
			return fmt.Sprintf("Error rating song: %v", err), true
		}
		return jsonText(object{"rating": rating}), false

	case "love":
		loved, err := musicAPI.ToggleLove()
		if err != nil {
			return fmt.Sprintf("Error toggling loved status: %v", err), true
		}
		return jsonText(object{"loved": loved}), false

	case "love_song":
		loved, err := musicAPI.SetLove(true)
		if err != nil {
			return fmt.Sprintf("Error loving song: %v", err), true
		}
		return jsonText(object{"loved": loved}), false

	case "dislike_song":
		if err := musicAPI.DislikeCurrentTrack(); err != nil {
			return fmt.Sprintf("Error disliking song: %v", err), true
		}
		return jsonText(object{"disliked": true}), false

	case "shuffle":
		enabled, err := musicAPI.ToggleShuffle()
		if err != nil {
			return fmt.Sprintf("Error toggling shuffle: %v", err), true
		}
		return jsonText(object{"shuffle": enabled}), false

	case "shuffle_mode":
		if enabled, ok := args["enabled"].(bool); ok {
			if err := musicAPI.SetShuffleMode(enabled); err != nil {
				return fmt.Sprintf("Error setting shuffle: %v", err), true
			}
			return jsonText(object{"shuffle": enabled}), false
		}
		enabled, err := musicAPI.ToggleShuffle()
		if err != nil {
			return fmt.Sprintf("Error toggling shuffle: %v", err), true
		}
		return jsonText(object{"shuffle": enabled}), false

	case "repeat":
		mode, err := musicAPI.CycleRepeat()
		if err != nil {
			return fmt.Sprintf("Error cycling repeat: %v", err), true
		}
		return jsonText(object{"repeat": mode}), false

	case "repeat_mode":
		mode, ok := args["mode"].(string)
		if !ok || mode == "" {
			return "mode argument is required", true
		}
		if err := musicAPI.SetRepeatMode(mode); err != nil {
			return fmt.Sprintf("Error setting repeat: %v", err), true
		}
		return jsonText(object{"repeat": mode}), false

	case "seek":
		if position, ok := args["position"].(float64); ok {
			if err := musicAPI.SetPlayerPosition(position); err != nil {
				return fmt.Sprintf("Error seeking: %v", err), true
			}
			return jsonText(object{"position": position}), false
		}
		seconds, ok := args["seconds"].(float64)
		if !ok || seconds == 0 {
			return "seconds must be a non-zero number, or position must be an absolute timestamp", true
		}
		if err := musicAPI.Seek(seconds); err != nil {
			return fmt.Sprintf("Error seeking: %v", err), true
		}
		return jsonText(object{"seek_seconds": seconds}), false

	case "get_volume":
		volume, err := musicAPI.GetVolume()
		if err != nil {
			return fmt.Sprintf("Error fetching volume: %v", err), true
		}
		return jsonText(object{"volume": volume}), false

	case "fade_out":
		seconds, _ := args["seconds"].(float64)
		if err := musicAPI.FadeOut(seconds); err != nil {
			return fmt.Sprintf("Error fading out: %v", err), true
		}
		return jsonText(object{"faded_out": true}), false

	case "get_recently_played":
		limit, _ := intArg(args, "limit")
		tracks, err := musicAPI.GetRecentlyPlayed(limit)
		if err != nil {
			return fmt.Sprintf("Error fetching recently played: %v", err), true
		}
		return jsonText(tracks), false

	case "top_tracks":
		limit, _ := intArg(args, "limit")
		tracks, err := musicAPI.GetTopTracks(limit)
		if err != nil {
			return fmt.Sprintf("Error fetching top tracks: %v", err), true
		}
		return jsonText(tracks), false

	case "add_to_library":
		if err := musicAPI.AddCurrentTrackToLibrary(); err != nil {
			return fmt.Sprintf("Error adding to library: %v", err), true
		}
		return "Current track added to library.", false

	case "get_recommendations":
		return "Apple Music For You recommendations are not exposed through Music.app AppleScript, so this MCP server cannot fetch them reliably.", true

	case "radio_mode":
		return "Apple Music station creation is not exposed reliably through Music.app AppleScript, so this MCP server cannot start radio mode yet.", true

	case "add_to_queue":
		return "Music.app AppleScript does not expose a reliable Up Next mutation API, so this MCP server cannot add songs to the queue yet.", true

		case "playlist_export":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return "name argument is required", true
		}
		format, _ := args["format"].(string)
		if format == "" {
			format = "m3u"
		}
		expContent, err := musicAPI.ExportPlaylist(name, format)
		if err != nil {
			return fmt.Sprintf("Error exporting playlist: %v", err), true
		}
		return expContent, false

	case "playlist_import":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return "name argument is required", true
		}
		pathOrContent, ok := args["path_or_content"].(string)
		if !ok || pathOrContent == "" {
			return "path_or_content argument is required", true
		}
		if err := musicAPI.ImportPlaylist(name, pathOrContent); err != nil {
			return fmt.Sprintf("Error importing playlist: %v", err), true
		}
		return fmt.Sprintf("Imported tracks into playlist: %s", name), false

	case "import_playlist":
		// Legacy alias: delegates to playlist_import
		path, ok := args["path"].(string)
		if !ok || path == "" {
			return "path argument is required", true
		}
		if err := musicAPI.ImportPlaylist("Imported Playlist", path); err != nil {
			return fmt.Sprintf("Error importing playlist: %v", err), true
		}
		return "Playlist imported as 'Imported Playlist'.", false

	case "music_context":
		return jsonText(buildMusicContext()), false

	case "dj_queue_add":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "query argument is required", true
		}
		desc, pos, err := music.DJAddToQueue(query)
		if err != nil {
			return fmt.Sprintf("%v", err), true
		}
		return fmt.Sprintf("Queued: %s (%d in queue)", desc, pos), false

	case "dj_queue_view":
		q := music.DJGetQueue()
		if len(q) == 0 {
			return "DJ queue is empty.", false
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "DJ Queue (%d tracks):\n", len(q))
		for i, e := range q {
			fmt.Fprintf(&sb, "%d. %s \u2013 %s\n", i+1, e.Title, e.Artist)
		}
		return strings.TrimRight(sb.String(), "\n"), false

	case "dj_queue_clear":
		if err := music.DJClearQueue(); err != nil {
			return fmt.Sprintf("Error clearing DJ queue: %v", err), true
		}
		return "DJ queue cleared.", false

	case "dj_skip":
		fadeSecs, _ := args["fade_seconds"].(float64)
		if fadeSecs <= 0 {
			fadeSecs = 3
		}
		if music.DjFading {
			return "Transition already in progress.", false
		}
		next, ok := music.DJDequeueNext()
		if !ok {
			return "DJ queue is empty.", true
		}
		music.DjFading = true
		origVol, err := musicAPI.GetVolume()
		if err != nil || origVol <= 0 {
			origVol = 80
		}
		musicAPI.FadeOut(fadeSecs)
		if err := musicAPI.PlayTrackByName(next.Title); err != nil {
			music.DjFading = false
			return fmt.Sprintf("Error playing next track: %v", err), true
		}
		musicAPI.FadeIn(fadeSecs, origVol)
		music.DjFading = false
		return fmt.Sprintf("Now playing: %s \u2013 %s", next.Title, next.Artist), false

	case "queue_add":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "query argument is required", true
		}
		desc, pos, err := music.DJAddToQueue(query)
		if err != nil {
			return fmt.Sprintf("%v", err), true
		}
		return fmt.Sprintf("Queued: %s (%d in queue)", desc, pos), false

	case "queue_clear":
		if err := music.DJClearQueue(); err != nil {
			return fmt.Sprintf("Error clearing queue: %v", err), true
		}
		return "Queue cleared.", false

	case "queue_move":
		from, ok := intArg(args, "from")
		if !ok {
			return "from argument is required and must be a positive integer", true
		}
		to, ok := intArg(args, "to")
		if !ok {
			return "to argument is required and must be a positive integer", true
		}
		if err := music.DJMoveQueueItem(from, to); err != nil {
			return fmt.Sprintf("Error moving queue item: %v", err), true
		}
		return fmt.Sprintf("Moved item from position %d to %d.", from, to), false

	default:
		return fmt.Sprintf("Unknown tool: %s", name), true
	}
}

func jsonText(v interface{}) string {
	bz, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(bz)
}

func intArg(args map[string]interface{}, name string) (int, bool) {
	raw, ok := args[name]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

func lyricsForCurrentTrack() ([]lyrics.Line, error) {
	info, err := musicAPI.NowPlaying()
	if err != nil {
		return nil, err
	}
	if info == nil || info.State == "stopped" {
		return []lyrics.Line{}, nil
	}
	lines, err := musicAPI.FetchLyrics(info.Title, info.Artist, info.Album, info.Duration)
	if err != nil {
		return nil, err
	}
	if lines == nil {
		lines = []lyrics.Line{}
	}
	return lines, nil
}

func buildMusicContext() object {
	ctx := object{}

	info, err := musicAPI.NowPlaying()
	if err != nil {
		ctx["now_playing_error"] = err.Error()
		return ctx
	}
	ctx["now_playing"] = info
	if info == nil || info.State == "stopped" {
		return ctx
	}

	if stats, err := musicAPI.GetTrackStats(); err == nil {
		ctx["stats"] = stats
	} else {
		ctx["stats_error"] = err.Error()
	}
	if queue, err := musicAPI.GetQueue(); err == nil {
		ctx["queue"] = queue
	} else {
		ctx["queue_error"] = err.Error()
	}
	if lines, err := musicAPI.FetchLyrics(info.Title, info.Artist, info.Album, info.Duration); err == nil {
		if lines == nil {
			lines = []lyrics.Line{}
		}
		ctx["lyrics"] = lines
	} else {
		ctx["lyrics_error"] = err.Error()
	}

	return ctx
}
