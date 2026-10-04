# Muse

Muse is a terminal controller for Apple Music on macOS. It provides a fast CLI, an interactive Bubble Tea dashboard, and an MCP stdio server for agent workflows.

[![CI](https://github.com/ryanrodrigues25200525-svg/Apple-music-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/ryanrodrigues25200525-svg/Apple-music-cli/actions/workflows/ci.yml)

![Muse dashboard](docs/assets/dashboard.svg)

## Highlights

- Control Music.app from the terminal with `mu`.
- Search and play tracks, playlists, queue, lyrics, artwork, love, shuffle, repeat, sleep timer, and stats.
- Launch a Bubble Tea dashboard with keyboard controls.
- Expose playback and library actions as MCP tools for agents.
- Built for macOS with AppleScript, no Apple Music API credentials required.

## Quick Start

```sh
brew tap ryanrodrigues25200525-svg/muse https://github.com/ryanrodrigues25200525-svg/Muse.git
brew install muse
mu doctor
mu
```

`mu` opens the interactive dashboard. Use `Space` to play or pause, `/` to search, and `q` to quit.

## Screenshots

| Dashboard | Playlists |
| --- | --- |
| ![Muse dashboard view](docs/assets/dashboard.svg) | ![Muse playlist browser](docs/assets/playlists.svg) |

| JSON and MCP |
| --- |
| ![Muse JSON and MCP output](docs/assets/mcp-json.svg) |

## Install

Muse requires macOS, Music.app, and AppleScript Automation permission for your terminal.

### Homebrew

```sh
brew tap ryanrodrigues25200525-svg/muse https://github.com/ryanrodrigues25200525-svg/Muse.git
brew install muse
```

Upgrade later with:

```sh
brew upgrade muse
```

### GitHub Releases

Download the latest macOS universal archive:

[Latest release](https://github.com/ryanrodrigues25200525-svg/Apple-music-cli/releases/latest)

Then unpack it and move `mu` into a directory on your `PATH`, such as `/usr/local/bin` or `~/.local/bin`.

### Install Script

From a checked-out repo:

```sh
./scripts/install.sh
```

Or directly from GitHub:

```sh
curl -fsSL https://raw.githubusercontent.com/ryanrodrigues25200525-svg/Apple-music-cli/main/scripts/install.sh | sh
```

### Build From Source

```sh
git clone https://github.com/ryanrodrigues25200525-svg/Apple-music-cli.git
cd Apple-music-cli
make build
./mu version
```

Install to `~/.local/bin`:

```sh
make install
```

## Commands

### Playback

```sh
mu play
mu play "Nights"
mu pause
mu toggle
mu next
mu prev
mu stop
mu seek +30
mu seek -10
```

### Library And Metadata

```sh
mu now
mu now --json
mu search "Frank Ocean"
mu search "Frank Ocean" --json
mu playlists
mu playlists --json
mu playlist "Gym"
mu playlist "Gym" --shuffle
mu queue
mu queue --json
mu stats
mu stats --json
mu love
```

### Playback Modes And Volume

```sh
mu volume
mu volume 50
mu shuffle toggle
mu repeat
mu sleep 30
```

### Interactive Modes

```sh
mu
mu mini
```

### Diagnostics And Shell Integration

```sh
mu doctor
mu completion zsh
mu completion bash
mu completion fish
```

## TUI

Running `mu` opens the dashboard.

Current keybindings:

- `Space`: toggle playback
- `n` / `p`: next / previous
- `s`: shuffle
- `r`: repeat
- `l`: love current track
- `[` / `]`: seek backward / forward
- `+` / `-`: volume up / down
- `/`: search
- `q`: quit

The dashboard includes now-playing details, progress, volume, shuffle/repeat/love state, playlist browsing, library search, lyrics, queue browsing, and a compact help footer.

## MCP

Start the MCP server over stdio:

```sh
mu mcp
```

Available tools include playback controls, playlist management (`list_playlists`, `get_playlist_tracks`, `jump_to_track`, `create_playlist`, `add_to_playlist`, `remove_from_playlist`), playback state (`now_playing`, `get_queue`, `get_volume`, `shuffle_mode`, `repeat_mode`, `seek`, `fade_out`), library metadata (`get_song_info`, `get_lyrics`, `get_album`, `rate_song`, `love_song`, `dislike_song`), history (`get_recently_played`, `top_tracks`), and `music_context`.

Some Apple Music surfaces are not reliably exposed through Music.app AppleScript. The MCP server advertises `get_recommendations`, `radio_mode`, `add_to_queue`, `add_to_library`, and `import_playlist` with explicit best-effort behavior or clear errors where the local scripting API cannot support the action.

Data-heavy tools return JSON text so agents can parse the response reliably.

Example tools:

- `now_playing`
- `music_context`
- `search`
- `play_song`
- `play_playlist_shuffled`
- `get_playlist_tracks`
- `get_lyrics`
- `get_queue`
- `seek`

## Troubleshooting

Run:

```sh
mu doctor
```

Common issues:

- Music.app is not installed or cannot be launched.
- `osascript` is unavailable.
- macOS Automation permission is missing. Open System Settings, Privacy & Security, Automation, then grant your terminal access to Music.app.
- Lyrics require network access to `lrclib.net`.

Useful checks:

```sh
mu doctor
mu now --json
mu playlists --json
```

## Privacy And Security

Muse does not require Apple Music API credentials. Local playback control happens through AppleScript and Music.app. Lyrics lookup sends the current track title, artist, album, and duration to `lrclib.net`.

Report security issues using [SECURITY.md](SECURITY.md).

## Release

Release builds inject the version with:

```sh
go build -ldflags "-X github.com/ryanrodrigues25200525-svg/Apple-music-cli/cmd.Version=vX.Y.Z" -o mu main.go
```

The included GoReleaser config builds macOS archives for Apple Silicon and Intel.

## License

MIT. See [LICENSE](LICENSE).
