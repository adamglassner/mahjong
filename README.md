# Mahjong: Jade Moon Garden

Mahjong solitaire in a moonlit Chinese garden, by Ironite. Written in Go with
[Ebitengine](https://ebitengine.org). Clear the board two matching tiles at a
time across twelve puzzles, from Easy to Hard. It ships on its own and in the
Ironite Launcher.

## How to play

- **Rules:** remove the tiles in matching pairs. A tile is free when nothing is on top of it and its left or right side is open. Identical tiles match; any Flower matches any Flower, and any Season matches any Season.
- **Help:** Hint shows an open pair. Shuffle mixes the remaining tiles and always leaves a board you can clear. Undo has no limit. Every deal can be cleared.
- **Stars:** three for a clear with no hints or shuffles, two for up to three, one otherwise. Best times are kept for each puzzle.
- **Keys:** `H` hint, `U`/`Ctrl+Z` undo, `S` shuffle, `N` new deal, `M` music, `Esc` back, `F11` full screen.

### Puzzles

| Easy | Medium | Hard |
|---|---|---|
| Lotus Pad (46) | Moon Gate (100) | Stone Turtle (144) |
| Stepping Stones (40) | Arched Bridge (100) | Jade Dragon (142) |
| Little Lantern (56) | Paper Fan (102) | Great Pagoda (142) |
| Koi (74) | Tea House (112) | Garden Fortress (142) |

### Theme

The game has three scenes: a Moon Garden with fireflies over the koi pond, a Lantern Festival street
with sky lanterns, and a Plum Blossom Dawn with falling petals. It also has three tile sets: Ivory & Jade, Celadon and Rosewood.
The One of Bamboo is a koi. The four tunes (*Moon Over the Koi Pond*,
*Lantern Walk*, *Plum Blossom Rain*, *Jade Pavilion*) use zither, bamboo flute and soft
strings, and the game synthesizes them live.

Saves and settings are kept in `%APPDATA%\Ironite\Mahjong\save.json`, outside the install folder.

## Updates

- **Standalone copy:** the game checks this repository's latest GitHub release when it starts (this can be turned off in Settings). When a newer version is out, it offers to install it. The game downloads `Mahjong-<version>-windows.zip` and checks it against GitHub's SHA-256 digest. A small script then swaps the files once the game closes and starts the new version. If the game is in Program Files, Windows asks for permission first.
- **Ironite Launcher:** the launcher installs and updates Mahjong from the same releases, and the game's own updater stays out of the way.

## Code

| File | What it is |
|---|---|
| `tiles.go`, `layouts.go` | The 144-tile set and matching rules; the twelve puzzle layouts. |
| `board.go`, `game.go` | Free tiles, solvable deals (built by playing the layout backwards), hints, shuffles, undo, stars. |
| `ui.go`, `main.go` | Screens, dialogs, input and the frame loop. |
| `tilegfx.go`, `scene.go`, `draw.go` | Tiles, scenes and UI drawn as vectors at the window's resolution. |
| `synth.go`, `music.go`, `audio.go` | The synthesizer, the four tunes and the sound effects. |
| `update.go` | GitHub release check and self-update (standalone copy). |
| `save.go` | The save file. |
| `tools/mkres/` | Writes the Windows icon and version info into the .exe. |
| `installer/` | Inno Setup scripts for the standalone `Mahjong-Setup-v<version>.exe`. |

## Tests

```bash
go test -tags headless .
```

The tests check the layouts (even tile counts, no overlaps, every raised tile supported), that every deal and every shuffle can be cleared, the hints, saving and restoring, the music, and the updater (version compare, download, checksum and zip safety).

## Building

You need Go 1.24 or newer and bash (Git Bash works on Windows):

```bash
./build.sh
```

That writes `dist/Games/Mahjong/` with `Mahjong.exe` and its text files. To make the installer, run `ISCC.exe installer/MahjongSetup.iss` (Inno Setup 6.3+).

## Releasing

Add a `## [x.y.z]` section to `CHANGELOG.md`. Set `Version` in `game.go` and `GameVersion` in `installer/MahjongSetup.iss` to the same number (build.sh checks this). Then push to `main`. The release workflow builds the game and publishes `Mahjong-<version>-windows.zip` as release `v<version>`. Ironite Launcher and the standalone copies pick it up from there.

(c) 2026 Ironite. Fonts: Young Serif, Work Sans, Noto Serif CJK (SIL Open Font License, `assets/fonts/OFL.txt`).
