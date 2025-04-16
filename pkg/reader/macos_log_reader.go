//go:build darwin
// +build darwin

package reader

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// macosExecCommand allows mocking exec.Command in tests
var macosExecCommand = exec.Command

// Initialize macos specific implementation
func init() {
	macosLogReaderFactory = func(query string) (LogReader, error) {
		return NewMacOSLogReader(query)
	}
}

// MacOSLogReader is a reader for macOS logs using the 'log' command
type MacOSLogReader struct {
	query     string
	lines     chan string
	stopCh    chan struct{}
	stoppedCh chan struct{}
	cmd       *exec.Cmd
	lock      sync.Mutex
	running   bool
}

// NewMacOSLogReader creates a new reader for macOS logs
func NewMacOSLogReader(query string) (*MacOSLogReader, error) {
	return &MacOSLogReader{
		query:     query,
		lines:     make(chan string, 1000),
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}, nil
}

// Start begins streaming logs from macOS log system
func (r *MacOSLogReader) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.running {
		return nil
	}

	// Build the command to stream logs
	args := []string{"stream"}

	// Add query if provided
	if r.query != "" {
		args = append(args, "--predicate", r.query)
	}

	// Add formatting options
	args = append(args, "--style", "syslog")

	// Create the command
	r.cmd = macosExecCommand("log", args...)

	// Get stdout pipe
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	// Start the command
	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start log command: %v", err)
	}

	r.running = true

	// Start the reader goroutine
	go r.readLogs(stdout)

	return nil
}

// readLogs is the main goroutine for reading log output
func (r *MacOSLogReader) readLogs(stdout io.ReadCloser) {
	defer func() {
		r.lock.Lock()
		if r.cmd != nil && r.cmd.Process != nil {
			// Send a SIGTERM signal to gracefully terminate the log command
			r.cmd.Process.Signal(syscall.SIGTERM)

			// Wait for the process to exit (with a timeout)
			done := make(chan error, 1)
			go func() {
				done <- r.cmd.Wait()
			}()

			select {
			case <-done:
				// Process exited
			case <-time.After(2 * time.Second):
				// Force kill if it doesn't exit
				r.cmd.Process.Kill()
			}

			r.cmd = nil
		}
		r.running = false
		r.lock.Unlock()
		close(r.stoppedCh)
	}()

	// Create a scanner to read the output line by line
	scanner := bufio.NewScanner(stdout)

	// Process each line
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Send the line to the channel
		select {
		case r.lines <- line:
		case <-r.stopCh:
			return
		}
	}

	// Check for errors
	if err := scanner.Err(); err != nil {
		// Only log the error if we didn't stop intentionally
		select {
		case <-r.stopCh:
			// We stopped intentionally, ignore the error
		default:
			errorMsg := fmt.Sprintf("Error reading macOS logs: %v", err)
			select {
			case r.lines <- errorMsg:
			default:
			}
		}
	}
}

// Lines returns the channel of log lines
func (r *MacOSLogReader) Lines() <-chan string {
	return r.lines
}

// Stop stops the reader and closes all resources
func (r *MacOSLogReader) Stop() {
	r.lock.Lock()
	if !r.running {
		r.lock.Unlock()
		return
	}
	r.lock.Unlock()

	close(r.stopCh)
	<-r.stoppedCh

	// Close the lines channel
	close(r.lines)
}
