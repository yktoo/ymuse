[![Latest release](https://img.shields.io/github/v/release/yktoo/ymuse.svg)](https://github.com/yktoo/ymuse/releases/latest)
[![License](https://img.shields.io/github/license/yktoo/ymuse.svg)](COPYING)
[![Build Status](https://travis-ci.org/yktoo/ymuse.svg?branch=master)](https://travis-ci.org/yktoo/ymuse)

# Ymuse

**Ymuse** is a GTK front-end (client) application for [Music Player Daemon](https://www.musicpd.org/) written in Go.

## Building from the source

### Requirements

* Go 1.14+

### Getting started

1. [Install Go](https://golang.org/doc/install)
2. [Install Mage](https://magefile.org/)
3. Clone the source and compile
```bash
git clone https://github.com/yktoo/ymuse.git
mage build
```

This will create the application executable `ymuse` in the project root directory, which you can run straight away.

## License

See [COPYING](COPYING)

## Credits

* [gotk3](https://github.com/gotk3/gotk3)
* [gompd](https://github.com/fhs/gompd) by Fazlul Shahriar
* [go-logging](https://github.com/op/go-logging) by Örjan Fors

