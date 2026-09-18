package main

import (
	"flag"
	"log"
	"os"

	"github.com/axsh/tokotachi/features/clock/internal/face"
	"github.com/axsh/tokotachi/features/clock/internal/winui"
)

func main() {
	quitAfter := flag.Duration("quit-after", 0, "auto-exit after duration (for tests); 0 disables")
	size := flag.Int("size", face.DefaultSize, "clock face size in pixels")
	flag.Parse()

	if err := winui.Run(winui.Options{Size: *size, QuitAfter: *quitAfter}); err != nil {
		log.Printf("ERROR: clock exited: %v", err)
		os.Exit(1)
	}
}
