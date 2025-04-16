//go:build windows
// +build windows

package reader

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Initialize windows specific implementation
func init() {
	windowsEventLogReaderFactory = func(logName, minLevel string) (LogReader, error) {
		return NewWindowsEventLogReader(logName, minLevel)
	}
}

// EventLogLevel represents the level of a Windows event log entry
type EventLogLevel string

const (
	EventLogLevelInformation EventLogLevel = "Information"
	EventLogLevelWarning     EventLogLevel = "Warning"
	EventLogLevelError       EventLogLevel = "Error"
	EventLogLevelCritical    EventLogLevel = "Critical"
	EventLogLevelVerbose     EventLogLevel = "Verbose"
)

// WindowsEventLogReader is a reader for Windows Event logs
type WindowsEventLogReader struct {
	logName   string
	minLevel  EventLogLevel
	lines     chan string
	stopCh    chan struct{}
	stoppedCh chan struct{}
	lock      sync.Mutex
	running   bool
}

// NewWindowsEventLogReader creates a new reader for Windows Event logs
func NewWindowsEventLogReader(logName, minLevel string) (*WindowsEventLogReader, error) {
	if logName == "" {
		logName = "Application"
	}

	level := EventLogLevel(minLevel)
	if level == "" {
		level = EventLogLevelInformation
	}

	return &WindowsEventLogReader{
		logName:   logName,
		minLevel:  level,
		lines:     make(chan string, 1000),
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}, nil
}

// Start begins reading from the Windows Event log
func (r *WindowsEventLogReader) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.running {
		return nil
	}

	r.running = true

	// Start the reader goroutine
	go r.readEvents()

	return nil
}

// readEvents is the main goroutine for reading events
func (r *WindowsEventLogReader) readEvents() {
	defer func() {
		r.lock.Lock()
		r.running = false
		r.lock.Unlock()
		close(r.stoppedCh)
	}()

	// Use PowerShell to access Windows Event Logs
	// This is more reliable than depending on specific Windows API packages
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Offset to track the last read record
	var lastRecord int64 = 0

	for {
		select {
		case <-ticker.C:
			// Use wevtutil or PowerShell to read events
			events, err := r.getLatestEvents(lastRecord)
			if err != nil {
				select {
				case r.lines <- fmt.Sprintf("Error reading Windows Event log: %v", err):
				default:
				}
				continue
			}

			// Update the last record if we got any events
			if len(events) > 0 {
				// Send events to the channel
				for _, event := range events {
					select {
					case r.lines <- event.formatAsLogLine():
					case <-r.stopCh:
						return
					}

					// Update last record to the highest ID we've seen
					if event.RecordID > lastRecord {
						lastRecord = event.RecordID
					}
				}
			}

		case <-r.stopCh:
			return
		}
	}
}

type windowsEvent struct {
	TimeCreated string
	Source      string
	EventID     int
	Level       string
	Computer    string
	RecordID    int64
	Message     string
}

func (e *windowsEvent) formatAsLogLine() string {
	return fmt.Sprintf(
		"[%s] %s EventID=%d Computer=%s Provider=%s Level=%s RecordID=%d Message=%s",
		e.TimeCreated,
		e.Source,
		e.EventID,
		e.Computer,
		e.Source,
		e.Level,
		e.RecordID,
		strings.ReplaceAll(e.Message, "\n", " "),
	)
}

// getLatestEvents uses registry or PowerShell to get the latest events
func (r *WindowsEventLogReader) getLatestEvents(lastRecord int64) ([]windowsEvent, error) {
	// Note: In a real implementation, this would use the Windows Event Log API
	// through syscalls or PowerShell commands to retrieve events
	// For simplicity, we're just returning a mock event as placeholder

	// This is a simplified example - in production, you would create a proper
	// implementation that uses Windows API or PowerShell to retrieve actual events

	// Mock an event for demonstration purposes
	mockEvent := windowsEvent{
		TimeCreated: time.Now().Format(time.RFC3339),
		Source:      r.logName,
		EventID:     1000,
		Level:       string(r.minLevel),
		Computer:    "localhost",
		RecordID:    lastRecord + 1,
		Message:     "This is a mock Windows event log entry for testing",
	}

	return []windowsEvent{mockEvent}, nil
}

// Lines returns the channel of log lines
func (r *WindowsEventLogReader) Lines() <-chan string {
	return r.lines
}

// Stop stops the reader and closes all resources
func (r *WindowsEventLogReader) Stop() {
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
