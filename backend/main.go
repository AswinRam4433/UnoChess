package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	"unochess/gameloop"
	"unochess/transport"
)

func main() {
	port := flag.String("port", "8080", "HTTP port for server mode")
	mode := flag.String("mode", "server", "server | bot")
	flag.Parse()

	switch *mode {
	case "server":
		runServer(*port)
	case "bot":
		runBot()
	default:
		log.Fatalf("unknown mode %q — use server or bot", *mode)
	}
}

func runServer(port string) {
	reg := transport.NewRegistry()
	srv := transport.NewServer(reg)
	addr := ":" + port
	log.Printf("UnoChess server listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}

func runBot() {
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
