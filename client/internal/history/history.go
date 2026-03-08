package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	cryptoutil "messenger-client/internal/crypto"
)

type record struct {
	Timestamp time.Time `json:"ts"`
	Sender    string    `json:"sender"`
	Payload   string    `json:"payload"`
}

type Entry struct {
	Timestamp time.Time
	Sender    string
	Message   string
}

func baseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".messenger", "history"), nil
}

func filePath(room string) (string, error) {
	dir, err := baseDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, room+".jsonl"), nil
}

func Append(room, key, sender, message string) error {
	path, err := filePath(room)
	if err != nil {
		return err
	}

	enc, err := cryptoutil.Encrypt(message, key)
	if err != nil {
		return err
	}

	rec := record{Timestamp: time.Now().UTC(), Sender: sender, Payload: enc}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

func Last(room, key string, n int) ([]Entry, error) {
	entries, err := readAll(room, key)
	if err != nil {
		return nil, err
	}
	if n <= 0 || n >= len(entries) {
		return entries, nil
	}
	return entries[len(entries)-n:], nil
}

func Export(room, key, outPath string) error {
	entries, err := readAll(room, key)
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, entry := range entries {
		line := fmt.Sprintf("%s [%s] %s\n", entry.Timestamp.Format(time.RFC3339), entry.Sender, entry.Message)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

func readAll(room, key string) ([]Entry, error) {
	path, err := filePath(room)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	entries := make([]Entry, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		msg, err := cryptoutil.Decrypt(rec.Payload, key)
		if err != nil {
			continue
		}
		entries = append(entries, Entry{Timestamp: rec.Timestamp, Sender: rec.Sender, Message: msg})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
