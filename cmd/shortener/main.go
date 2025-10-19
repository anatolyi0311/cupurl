package main

import (
	"github.com/anatolyi0311/cupurl/internal/server"
)

func main() {
	s := server.NewServer()
	s.Run()
}
