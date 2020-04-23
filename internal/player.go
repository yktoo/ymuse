package internal

import (
	"github.com/fhs/gompd/v2/mpd"
	"github.com/pkg/errors"
)

type Player struct {
	client *mpd.Client
}

func NewPlayer(address string) (Player, error) {
	player := Player{}
	err := player.connect(address)
	return player, err
}

func (player *Player) verifyConnected() error {
	if player.client == nil {
		return errors.New("Not connected to MPD")
	}
	return nil
}

func (player *Player) connect(address string) error {
	client, err := mpd.Dial("tcp", address)
	player.client = client
	return err
}

func (player *Player) Stats() (mpd.Attrs, error) {
	if err := player.verifyConnected(); err != nil {
		return nil, err
	}
	return player.client.Stats()
}

func (player *Player) Status() (mpd.Attrs, error) {
	if err := player.verifyConnected(); err != nil {
		return nil, err
	}
	return player.client.Status()
}

func (player *Player) Version() (string, error) {
	if err := player.verifyConnected(); err != nil {
		return "", err
	}
	return player.client.Version(), nil
}
