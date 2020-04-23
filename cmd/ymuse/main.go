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

	// Print MPD version
	ver, err := player.Version()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Connected to MPD version %v\n", ver)

	// Print out status
	status, err := player.Status()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("MPD status:")
	for k, v := range status {
		fmt.Printf("  - %v: %v\n", k, v)
	}

	// Print out statistics
	stats, err := player.Stats()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("MPD database statistics:")
	for k, v := range stats {
		fmt.Printf("  - %v: %v\n", k, v)
	}
}
