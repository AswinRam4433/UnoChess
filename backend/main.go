package main

import (
	"fmt"
	"math/rand/v2"
	"time"

	"unochess/gameloop"
)

// main runs a single bot-driven UnoChess game and prints the result. The seed is
// taken from the wall clock so each run is fresh; the seed is printed up-front so a
// specific run can be reproduced by hard-coding it.
func main() {
	seed := uint64(time.Now().UnixNano())
	fmt.Printf("seed=%d\n", seed)

	rng := rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))
	g := gameloop.NewUnoChessGameWith(rng)

	res, err := gameloop.RunGame(g, gameloop.RunOptions{})
	if err != nil {
		fmt.Printf("Game errored: %v\n", err)
		return
	}
	fmt.Printf("Game over after %d turns: %s (winner: %v)\n", res.Turns, res.Reason, res.Winner)
}
