[![Latest release](https://img.shields.io/github/v/release/yktoo/ymuse.svg)](https://github.com/yktoo/ymuse/releases/latest)
[![Releases](https://img.shields.io/github/downloads/yktoo/ymuse/total.svg)](https://github.com/yktoo/ymuse/releases)
[![License](https://img.shields.io/github/license/yktoo/ymuse.svg)](COPYING)
[![Go](https://github.com/yktoo/ymuse/actions/workflows/go.yml/badge.svg)](https://github.com/yktoo/ymuse/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/yktoo/ymuse)](https://goreportcard.com/report/github.com/yktoo/ymuse)

# ![Ymuse icon](resources/icons/hicolor/32x32/apps/ymuse.png) Ymuse

**Ymuse** is an easy, functional, and snappy GTK front-end (client) for [Music Player Daemon](https://www.musicpd.org/) written in Go. It supports both light and dark desktop theme.

[![Ymuse screenshot](https://res.cloudinary.com/yktoo/image/upload/blog/e6ecokfftenpwlwswon1.png)](https://res.cloudinary.com/yktoo/image/upload/blog/e6ecokfftenpwlwswon1.png)

It supports library browsing and search, playlists, streams etc.

[![Ymuse Library screenshot](https://res.cloudinary.com/yktoo/image/upload/t_s320/blog/wqud8spomcmuduvgar9d.png)](https://res.cloudinary.com/yktoo/image/upload/blog/wqud8spomcmuduvgar9d.png)
[![Ymuse Streams screenshot](https://res.cloudinary.com/yktoo/image/upload/t_s320/blog/pnwj9nlucfuobw0vcv0l.png)](https://res.cloudinary.com/yktoo/image/upload/blog/pnwj9nlucfuobw0vcv0l.png)

Watch Ymuse feature tour video:

[![Feature tour video](https://img.youtube.com/vi/h0g2gk5DM8s/0.jpg)](https://www.youtube.com/watch?v=h0g2gk5DM8s)

## Installing

You can:

* If your distribution supports [snap packages](https://snapcraft.io/ymuse): `sudo snap install ymuse`
* Otherwise, you can use a binary package from the [Releases](https://github.com/yktoo/ymuse/releases) section.

## Building from source

### Requirements

* Go 1.18+
* GTK 3.22+

### Getting started

1. [Install Go](https://golang.org/doc/install)
2. Make sure you have the following build dependencies installed:
   * `build-essential`
   * `libc6`
   * `libgtk-3-dev`
   * `libgdk-pixbuf2.0-dev`
   * `libglib2.0-dev`
   * `gettext`
3. Clone the source and compile:
```bash
git clone https://github.com/yktoo/ymuse.git
cd ymuse
go generate
go build
```
4. Copy over the icons and localisations:
```bash
sudo cp -r resources/icons/* /usr/share/icons/
sudo cp -r resources/i18n/generated/* /usr/share/locale/
sudo update-icon-caches /usr/share/icons/hicolor/*
```

This will create the application executable `ymuse` in the project root directory, which you can run straight away.

## Packaging

### DEB and RPM

Requires `goreleaser` installed.

```bash
goreleaser release --rm-dist --skip-publish [--snapshot]
```

### Flatpak

1. Install `flatpak` and `flatpack-builder`
2. `flatpak remote-add flathub https://flathub.org/repo/flathub.flatpakrepo`
3. `flatpak-builder dist /path/to/com.yktoo.ymuse.yml --force-clean --install-deps-from=flathub --repo=/path/to/repository`
4. Optional: make a `.flatpak` bundle:
   `flatpak build-bundle /path/to/repository ymuse.flatpak com.yktoo.ymuse`

## License

See [COPYING](COPYING).

## Credits

* [gotk3](https://github.com/gotk3/gotk3)
* [gompd](https://github.com/fhs/gompd) by Fazlul Shahriar
* [go-logging](https://github.com/op/go-logging) by Örjan Fors
* [goreleaser](https://goreleaser.com/) by Carlos Alexandro Becker et al.

## TODO

* Automated UI testing.
* Drag’n’drop in the play queue.
* More settings.
* Multiple MPD connections support.
