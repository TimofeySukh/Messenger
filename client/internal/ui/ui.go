package ui

import (
	"fmt"
	"io"
)

func PrintBanner(w io.Writer) {
	fmt.Fprintln(w, "Terminal Messenger")
	fmt.Fprintln(w, "Type /help for available commands")
}

func PrintSessionInfo(w io.Writer, username, room, key string) {
	fmt.Fprintf(w, "Connected as %s\n", username)
	fmt.Fprintf(w, "Room: %s\n", room)
	fmt.Fprintf(w, "Encryption key: %s\n", key)
}
