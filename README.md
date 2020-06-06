[![Latest release](https://img.shields.io/github/v/release/yktoo/ymuse.svg)](https://github.com/yktoo/ymuse/releases/latest)
[![License](https://img.shields.io/github/license/yktoo/ymuse.svg)](COPYING)
[![Build Status](https://travis-ci.org/yktoo/ymuse.svg?branch=master)](https://travis-ci.org/yktoo/ymuse)
[![Go Report Card](https://goreportcard.com/badge/github.com/yktoo/ymuse)](https://goreportcard.com/report/github.com/yktoo/ymuse)

# ![Ymuse icon](resources/icons/hicolor/32x32/apps/ymuse.png) Ymuse

**Ymuse** is an easy, functional, and snappy GTK front-end (client) for [Music Player Daemon](https://www.musicpd.org/) written in Go.

[![Ymuse screenshot](https://res.cloudinary.com/yktoo/image/upload/blog/jskaqgrbxzjyi7ofxetn.png)](https://res.cloudinary.com/yktoo/image/upload/blog/jskaqgrbxzjyi7ofxetn.png)

It supports library browsing and search, playlists, streams etc.

[![Ymuse screenshot](https://res.cloudinary.com/yktoo/image/upload/blog/zqu4ugqg0bvlh2hvajst.png)](https://res.cloudinary.com/yktoo/image/upload/blog/zqu4ugqg0bvlh2hvajst.png)

## Installing

You can use one of the binary packages from the [Releases](https://github.com/yktoo/ymuse/releases) section.

## Building from the source

### Requirements

* Go 1.14+

### Getting started

1. [Install Go](https://golang.org/doc/install)
2. Clone the source and compile
```bash
git clone https://github.com/yktoo/ymuse.git
go generate
go build
```

This will create the application executable `ymuse` in the project root directory, which you can run straight away.

## License

See [COPYING](COPYING).

## Credits

* [gotk3](https://github.com/gotk3/gotk3)
* [gompd](https://github.com/fhs/gompd) by Fazlul Shahriar
* [go-logging](https://github.com/op/go-logging) by Örjan Fors
* [goreleaser](https://goreleaser.com/) by Carlos Alexandro Becker et al.

