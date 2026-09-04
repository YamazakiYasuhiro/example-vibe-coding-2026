package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	format := flag.String("format", "iso", "date format: iso, slash, or jp")
	flag.Parse()

	dateStr, err := FormatToday(time.Now(), *format)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("Hello, World!")
	fmt.Println(dateStr)
}
