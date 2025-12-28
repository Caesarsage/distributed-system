package chatroom

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WAL - Write-ahead-log

func (cr *ChatRoom) initializePersistence() error {
	if err := os.MkdirAll(cr.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	walPath := filepath.Join(cr.dataDir, "messages.wal")

	if err := cr.recoverFromWAL(walPath); err != nil {
		fmt.Printf("Recovery failed: %v\n", err)
	}

	file, err := os.OpenFile(walPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open wal: %w", err)
	}

	cr.walFile = file
	fmt.Printf("WAL initialized: %s\n", walPath)
	return nil
}

func (cr *ChatRoom) recoverFromWAL(walPath string) error {
	file, err := os.Open(walPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No WAL file found (fresh start)")
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	recovered := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Printf("Skipping corrupt line: %s\n", line)
			continue
		}
		cr.messages = append(cr.messages, msg)
		if msg.ID >= cr.nextMessageID {
			cr.nextMessageID = msg.ID + 1
		}
		recovered++
	}

	fmt.Printf("Recovered %d messages \n", recovered)
	return nil
}

func (cr *ChatRoom) persistMessage(msg Message) error {
	cr.messageMu.Lock()
	defer cr.messageMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = cr.walFile.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	return cr.walFile.Sync()
}

func (cr *ChatRoom) createSnapshot() error {
	snapshortPath := filepath.Join(cr.dataDir, "snapshot.json")
	tempPath := snapshortPath + ".tmp"

	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer file.Close()

	cr.messageMu.Lock()
	data, err := json.MarshalIndent(cr.messages, "", "  ")
	cr.messageMu.Unlock()

	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		return err
	}

	if err := file.Sync(); err != nil {
		return err
	}

	file.Close()
	if err := os.Rename(tempPath, snapshortPath); err != nil {
		return err
	}

	fmt.Printf("Snapshot created (%d messages)\n", len(cr.messages))
	return cr.truncateWAL()
}

func (cr *ChatRoom) truncateWAL() error {
	cr.walMu.Lock()
	defer cr.walMu.Unlock()

	if cr.walFile != nil {
		cr.walFile.Close()
	}

	walPath := filepath.Join(cr.dataDir, "messages.wal")
	file, err := os.OpenFile(walPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	cr.walFile = file
	fmt.Println(" WAL truncated")
	return nil
}

func (cr *ChatRoom) loadSnapshot() error {
	snapshotPath := filepath.Join(cr.dataDir, "snapshot.json")
	file, err := os.Open(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	cr.messageMu.Lock()
	err = json.Unmarshal(data, &cr.messages)
	cr.messageMu.Unlock()
	if err != nil {
		return err
	}

	for _, msg := range cr.messages {
		if msg.ID >= cr.nextMessageID {
			cr.nextMessageID = msg.ID + 1
		}
	}

	fmt.Printf("Loaded %d messages from snapshot\n", len(cr.messages))
	return nil
}
