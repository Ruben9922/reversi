# Reversi

[![ruben-reversi](https://snapcraft.io/ruben-reversi/badge.svg)](https://snapcraft.io/ruben-reversi)
[![goreleaser](https://github.com/Ruben9922/reversi/actions/workflows/release.yml/badge.svg)](https://github.com/Ruben9922/reversi/actions/workflows/release.yml)
[![GitHub](https://img.shields.io/github/license/Ruben9922/reversi)](https://github.com/Ruben9922/reversi/blob/master/LICENSE)

Command-line version of the classic Reversi / Othello game.

It supports both the modern Othello rules and the historical Reversi rules. The rules can be changed by pressing <kbd>R</kbd> on the title screen. Some info on the differences can be found [here](https://www.mastersofgames.com/rules/reversi-othello-rules.htm) and [here](https://en.wikipedia.org/wiki/Reversi#Rules).

[![asciicast](https://asciinema.org/a/mGiPozcB9NhEpVsh9CwQWsA52.svg)](https://asciinema.org/a/mGiPozcB9NhEpVsh9CwQWsA52)

## Usage

### Using a binary
Download the latest binary for your OS and architecture from the [releases page](https://github.com/Ruben9922/reversi/releases). Simply extract and run it; no installation needed.

#### Windows
1. Extract the zip archive.
2. Navigate into the folder where you extracted the files.
3. Run `reversi.exe`.

#### Linux or macOS
Extract the tar.gz archive using a GUI tool or the command line, e.g.:
```bash
tar -xvzf reversi_0.1.1_linux_x86_64.tar.gz --one-top-level
```

Navigate into the folder where you extracted the files, e.g.:
```bash
cd reversi_0.1.1_linux_x86_64/
```

Run the program:
```bash
./reversi
```

### Using Snap (Linux or macOS only)
If using Linux or macOS (with Snap installed), you can install via Snap using either the desktop store or the command line:
```bash
sudo snap install ruben-reversi
```

Run the game using the following command:
```bash
ruben-reversi
```
