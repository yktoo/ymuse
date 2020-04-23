package main

import (
	"fmt"
	"github.com/yktoo/ymuse/internal"
	"log"
)

const version = "0.01"

func main() {
	fmt.Printf("Ymuse version %s\n", version)

	// Instantiate a player
	player, err := internal.NewPlayer("127.0.0.1:6600")
	if err != nil {
		log.Fatalln(err)
	}

	ver, err := player.Version()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Connected to MPD version %v", ver)
}
