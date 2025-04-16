package reader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// FileReader represents a component that tails a log file
type FileReader struct {
	path           string
	file           *os.File
	reader         *bufio.Reader
	offset         int64
	lock           sync.Mutex
	lines          chan string
	stopCh         chan struct{}
	stoppedCh      chan struct{}
	reopenInterval time.Duration
}

// NewFileReader creates a new file reader
func NewFileReader(path string) *FileReader {
	return &FileReader{
		path:           path,
		lines:          make(chan string, 1000),
		stopCh:         make(chan struct{}),
		stoppedCh:      make(chan struct{}),
		reopenInterval: 1 * time.Second,
	}
}

// Start begins the log tailing process
func (r *FileReader) Start() error {
	var err error
	r.lock.Lock()
	r.file, err = os.Open(r.path)
	if err != nil {
		r.lock.Unlock()
		return fmt.Errorf("error opening file: %v", err)
	}

	// Seek to the end of the file for initial reading
	r.offset, err = r.file.Seek(0, io.SeekEnd)
	if err != nil {
		r.file.Close()
		r.lock.Unlock()
		return fmt.Errorf("error seeking file: %v", err)
	}

	r.reader = bufio.NewReader(r.file)
	r.lock.Unlock()

	go r.tailFile()
	return nil
}

// Lines returns the channel of log lines
func (r *FileReader) Lines() <-chan string {
	return r.lines
}

// Stop stops the file reader
func (r *FileReader) Stop() {
	close(r.stopCh)
	<-r.stoppedCh
}

// tailFile continuously reads the file and sends lines to the channel
func (r *FileReader) tailFile() {
	defer func() {
		r.lock.Lock()
		if r.file != nil {
			r.file.Close()
			r.file = nil
		}
		r.lock.Unlock()
		close(r.stoppedCh)
	}()

	for {
		select {
		case <-r.stopCh:
			return
		default:
			line, err := r.readLine()
			if err != nil {
				// If file was rotated or removed, attempt to reopen it
				time.Sleep(r.reopenInterval)
				r.reopen()
				continue
			}

			if line != "" {
				r.lines <- line
			} else {
				// No new line available, sleep briefly
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// readLine reads a single line from the file
func (r *FileReader) readLine() (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.file == nil {
		return "", fmt.Errorf("file is closed")
	}

	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Update offset if we successfully read a line
	r.offset += int64(len(line))

	// Trim the newline character
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	return line, nil
}

// reopen attempts to reopen the file, handling log rotation
func (r *FileReader) reopen() {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Close the existing file if it's open
	if r.file != nil {
		r.file.Close()
		r.file = nil
	}

	// Attempt to reopen the file
	var err error
	r.file, err = os.Open(r.path)
	if err != nil {
		// File might not exist yet, we'll retry later
		return
	}

	// Check if the file is a new one (e.g., after rotation)
	info, err := r.file.Stat()
	if err != nil {
		r.file.Close()
		r.file = nil
		return
	}

	// If the file is smaller than our last offset, it's likely a new file
	if info.Size() < r.offset {
		r.offset = 0
	}

	// Seek to the appropriate position
	_, err = r.file.Seek(r.offset, io.SeekStart)
	if err != nil {
		r.file.Close()
		r.file = nil
		return
	}

	r.reader = bufio.NewReader(r.file)
}
