package reader

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFileReader_Start(t *testing.T) {
	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "reader-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "test.log")
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}

	// Write initial content
	initialContent := "existing line 1\nexisting line 2\n"
	if _, err := file.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	file.Close()

	// Create a file reader
	reader := NewFileReader(logFile)
	if err := reader.Start(); err != nil {
		t.Fatalf("Failed to start reader: %v", err)
	}
	defer reader.Stop()

	// Append new lines to the log file
	time.Sleep(100 * time.Millisecond) // Give the reader time to start
	file, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file for appending: %v", err)
	}

	newLines := "new line 1\nnew line 2\nnew line 3\n"
	if _, err := file.WriteString(newLines); err != nil {
		t.Fatalf("Failed to append to log file: %v", err)
	}
	file.Close()

	// Read lines from the channel
	var readLines []string
	timeout := time.After(2 * time.Second)
	expectedLineCount := 3 // We expect to read the 3 new lines

	for len(readLines) < expectedLineCount {
		select {
		case line := <-reader.Lines():
			readLines = append(readLines, line)
		case <-timeout:
			t.Fatalf("Timeout waiting for lines. Got %d lines, expected %d", len(readLines), expectedLineCount)
			return
		}
	}

	// Verify the read lines
	for i, expectedLine := range []string{"new line 1", "new line 2", "new line 3"} {
		if i >= len(readLines) {
			t.Fatalf("Missing expected line: %s", expectedLine)
		}
		if readLines[i] != expectedLine {
			t.Errorf("Expected line %d to be '%s', got '%s'", i, expectedLine, readLines[i])
		}
	}
}

func TestFileReader_Rotation(t *testing.T) {
	// Skip test on Windows as file handles behave differently
	if runtime.GOOS == "windows" {
		t.Skip("Skipping rotation test on Windows")
	}

	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "rotation-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "rotate.log")
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}

	// Write initial content
	if _, err := file.WriteString("original line 1\noriginal line 2\n"); err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	file.Close()

	// Create a file reader
	reader := NewFileReader(logFile)
	if err := reader.Start(); err != nil {
		t.Fatalf("Failed to start reader: %v", err)
	}
	defer reader.Stop()

	// Wait a bit for the reader to initialize
	time.Sleep(100 * time.Millisecond)

	// Append new content to original file
	file, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file for appending: %v", err)
	}
	if _, err := file.WriteString("pre-rotation line\n"); err != nil {
		t.Fatalf("Failed to append to log file: %v", err)
	}
	file.Close()

	// Read lines until we get the pre-rotation line
	foundPreRotation := false
	timeout := time.After(2 * time.Second)
readPreRotation:
	for {
		select {
		case line := <-reader.Lines():
			if line == "pre-rotation line" {
				foundPreRotation = true
				break readPreRotation
			}
		case <-timeout:
			t.Fatal("Timeout waiting for pre-rotation line")
			return
		}
	}

	if !foundPreRotation {
		t.Fatal("Did not receive pre-rotation line")
	}

	// Simulate log rotation by renaming the file and creating a new one
	rotatedFile := filepath.Join(tempDir, "rotate.log.1")
	if err := os.Rename(logFile, rotatedFile); err != nil {
		t.Fatalf("Failed to rename log file: %v", err)
	}

	// Create a new log file
	file, err = os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create new log file: %v", err)
	}
	if _, err := file.WriteString("post-rotation line 1\npost-rotation line 2\n"); err != nil {
		t.Fatalf("Failed to write to new log file: %v", err)
	}
	file.Close()

	// Read lines after rotation
	var postRotationLines []string
	timeout = time.After(3 * time.Second)
	expectedPostRotationLines := 2

	for len(postRotationLines) < expectedPostRotationLines {
		select {
		case line := <-reader.Lines():
			if strings.HasPrefix(line, "post-rotation") {
				postRotationLines = append(postRotationLines, line)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for post-rotation lines. Got %d lines, expected %d",
				len(postRotationLines), expectedPostRotationLines)
			return
		}
	}

	// Verify post-rotation lines
	expectedLines := []string{"post-rotation line 1", "post-rotation line 2"}
	for i, expected := range expectedLines {
		if i >= len(postRotationLines) {
			t.Fatalf("Missing post-rotation line: %s", expected)
		}
		if postRotationLines[i] != expected {
			t.Errorf("Expected post-rotation line %d to be '%s', got '%s'",
				i, expected, postRotationLines[i])
		}
	}
}

// TestFileReader_Lines tests the Lines method
func TestFileReader_Lines(t *testing.T) {
	reader := NewFileReader("test-path.log")

	// Test that Lines() returns the expected channel
	lines := reader.Lines()
	if lines != reader.lines {
		t.Errorf("Lines() did not return the expected channel")
	}
}

// TestFileReader_StartStop tests the Start and Stop methods including proper cleanup
func TestFileReader_StartStop(t *testing.T) {
	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "start-stop-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "start-stop.log")
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	file.Close()

	// Create and start a file reader
	reader := NewFileReader(logFile)
	if err := reader.Start(); err != nil {
		t.Fatalf("Failed to start reader: %v", err)
	}

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop the reader and ensure it completes
	stopCompleteCh := make(chan struct{})
	go func() {
		reader.Stop()
		close(stopCompleteCh)
	}()

	// Wait for Stop to complete with timeout
	select {
	case <-stopCompleteCh:
		// Stop completed successfully
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout waiting for reader to stop")
	}

	// Verify the reader has properly cleaned up resources
	reader.lock.Lock()
	if reader.file != nil {
		t.Errorf("File was not closed after Stop()")
	}
	reader.lock.Unlock()
}

// TestFileReader_NonExistentFile tests handling of a non-existent file
func TestFileReader_NonExistentFile(t *testing.T) {
	nonExistentPath := "/path/does/not/exist/test.log"
	reader := NewFileReader(nonExistentPath)

	err := reader.Start()
	if err == nil {
		t.Errorf("Expected error when starting reader with non-existent file, got nil")
		reader.Stop() // Clean up if test fails
	}
}

// TestFileReader_Reopen tests the reopen method
func TestFileReader_Reopen(t *testing.T) {
	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "reopen-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "reopen.log")

	// Don't create the file initially

	// Create a file reader
	reader := NewFileReader(logFile)

	// Manually initialize some fields for testing reopen
	reader.offset = 100 // Set a non-zero offset

	// Reopen should handle non-existent file gracefully
	reader.reopen()

	reader.lock.Lock()
	if reader.file != nil {
		t.Errorf("Expected nil file after reopening non-existent file")
	}
	reader.lock.Unlock()

	// Now create the file
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}

	content := "line 1\nline 2\nline 3\n"
	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	file.Close()

	// Reopen should now succeed and reset offset since file is smaller
	reader.reopen()

	reader.lock.Lock()
	if reader.file == nil {
		t.Errorf("File is nil after reopening existing file")
	}

	if reader.offset != 0 {
		t.Errorf("Offset not reset after reopening smaller file: got %d, expected 0", reader.offset)
	}
	reader.lock.Unlock()

	// Clean up
	if reader.file != nil {
		reader.lock.Lock()
		reader.file.Close()
		reader.file = nil
		reader.lock.Unlock()
	}
}

// --- NEW TESTS BELOW ---

// TestFileReader_PermissionDenied tests handling of a file with no permission
func TestFileReader_PermissionDenied(t *testing.T) {
	// Skip on Windows where permissions work differently
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "permission-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "noperm.log")
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	file.Close()

	// Remove read permissions
	if err := os.Chmod(logFile, 0o000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Try to start reader
	reader := NewFileReader(logFile)
	err = reader.Start()

	// Should get permission denied error
	if err == nil {
		t.Errorf("Expected permission error, got nil")
		reader.Stop() // Clean up if test fails
	} else if !strings.Contains(err.Error(), "permission") &&
		!strings.Contains(err.Error(), "denied") &&
		!strings.Contains(err.Error(), "access") {
		t.Errorf("Expected permission error, got: %v", err)
	}
}

// TestFileReader_DynamicFileCreation tests handling of files that don't exist initially but are created later
func TestFileReader_DynamicFileCreation(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "dynamic-creation-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Path to a file that doesn't exist yet
	logFile := filepath.Join(tempDir, "future.log")

	// Create reader for a file that doesn't exist yet
	reader := NewFileReader(logFile)

	// Override reopenInterval to make the test faster
	reader.reopenInterval = 100 * time.Millisecond

	// Start should fail initially
	err = reader.Start()
	if err == nil {
		t.Errorf("Expected error when starting reader with non-existent file, got nil")
	}

	// Create the file after a delay
	time.Sleep(200 * time.Millisecond)
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// Write some content
	_, err = file.WriteString("dynamic line 1\ndynamic line 2\n")
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	file.Close()

	// Now manually reopen and start reading
	reader = NewFileReader(logFile)
	if err := reader.Start(); err != nil {
		t.Fatalf("Failed to start reader: %v", err)
	}
	defer reader.Stop()

	// Append more content
	time.Sleep(200 * time.Millisecond)
	file, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file for appending: %v", err)
	}
	_, err = file.WriteString("dynamic line 3\n")
	if err != nil {
		t.Fatalf("Failed to append to log file: %v", err)
	}
	file.Close()

	// Read lines from the channel
	var readLines []string
	timeout := time.After(1 * time.Second)
	expectedLine := "dynamic line 3"

	foundLine := false
	for !foundLine {
		select {
		case line := <-reader.Lines():
			readLines = append(readLines, line)
			if line == expectedLine {
				foundLine = true
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for line: %s. Got lines: %v", expectedLine, readLines)
			return
		}
	}
}

// TestFileReader_PartialLines tests handling of lines without trailing newlines
func TestFileReader_PartialLines(t *testing.T) {
	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "partial-line-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "partial.log")
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}

	// Create an empty file first
	file.Close()

	// Create a file reader and start it before writing anything
	reader := NewFileReader(logFile)
	if err := reader.Start(); err != nil {
		t.Fatalf("Failed to start reader: %v", err)
	}
	defer reader.Stop()

	// Sleep to allow reader to initialize
	time.Sleep(100 * time.Millisecond)

	// Now write the initial content
	file, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file for appending: %v", err)
	}
	_, err = file.WriteString("complete line\npartial")
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	file.Close()

	// Sleep to give the reader time to process
	time.Sleep(100 * time.Millisecond)

	// Complete the partial line
	file, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file for appending: %v", err)
	}
	_, err = file.WriteString(" line completed\n")
	if err != nil {
		t.Fatalf("Failed to append to log file: %v", err)
	}
	file.Close()

	// Read lines from the channel
	var readLines []string
	timeout := time.After(3 * time.Second)
	expectedLineCount := 2 // One complete line and one partial line

	for len(readLines) < expectedLineCount {
		select {
		case line := <-reader.Lines():
			readLines = append(readLines, line)
			t.Logf("Read line: %s", line) // Log the line for debugging
		case <-timeout:
			t.Fatalf("Timeout waiting for lines. Got %d lines, expected %d: %v",
				len(readLines), expectedLineCount, readLines)
			return
		}
	}

	// Verify the complete line and completed partial line
	expected := []string{"complete line", "partial line completed"}
	for i, exp := range expected {
		if i >= len(readLines) {
			t.Fatalf("Missing expected line: %s", exp)
		}
		if readLines[i] != exp {
			t.Errorf("Line %d: expected '%s', got '%s'", i, exp, readLines[i])
		}
	}
}

// TestFileReader_FileRemovalAndRecreation tests handling of file being removed and recreated
func TestFileReader_FileRemovalAndRecreation(t *testing.T) {
	// Skip on Windows where file removal behavior is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping file removal test on Windows")
	}

	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "removal-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "removal.log")
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}

	// Write initial content
	_, err = file.WriteString("initial line 1\ninitial line 2\n")
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	file.Close()

	// Create a reader with shorter reopen interval for testing
	reader := NewFileReader(logFile)
	reader.reopenInterval = 200 * time.Millisecond

	if err := reader.Start(); err != nil {
		t.Fatalf("Failed to start reader: %v", err)
	}
	defer reader.Stop()

	// Wait for reader to initialize
	time.Sleep(100 * time.Millisecond)

	// Add a line before removal
	file, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file for appending: %v", err)
	}
	_, err = file.WriteString("pre-removal line\n")
	if err != nil {
		t.Fatalf("Failed to append to log file: %v", err)
	}
	file.Close()

	// Wait to read the pre-removal line
	foundPreRemoval := false
	timeout := time.After(1 * time.Second)
	for !foundPreRemoval {
		select {
		case line := <-reader.Lines():
			if line == "pre-removal line" {
				foundPreRemoval = true
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for pre-removal line")
			return
		}
	}

	// Remove the file
	if err := os.Remove(logFile); err != nil {
		t.Fatalf("Failed to remove log file: %v", err)
	}

	// Wait a bit
	time.Sleep(300 * time.Millisecond)

	// Recreate the file
	file, err = os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to recreate log file: %v", err)
	}
	_, err = file.WriteString("post-removal line 1\npost-removal line 2\n")
	if err != nil {
		t.Fatalf("Failed to write to recreated log file: %v", err)
	}
	file.Close()

	// Read lines after file recreation
	var postRemovalLines []string
	timeout = time.After(3 * time.Second)
	expectedPostRemovalLines := 2

	for len(postRemovalLines) < expectedPostRemovalLines {
		select {
		case line := <-reader.Lines():
			if strings.HasPrefix(line, "post-removal") {
				postRemovalLines = append(postRemovalLines, line)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for post-removal lines. Got %d lines, expected %d: %v",
				len(postRemovalLines), expectedPostRemovalLines, postRemovalLines)
			return
		}
	}

	// Verify post-removal lines
	expected := []string{"post-removal line 1", "post-removal line 2"}
	for i, exp := range expected {
		if i >= len(postRemovalLines) {
			t.Fatalf("Missing post-removal line: %s", exp)
		}
		if postRemovalLines[i] != exp {
			t.Errorf("Post-removal line %d: expected '%s', got '%s'", i, exp, postRemovalLines[i])
		}
	}
}

// TestFileReader_ReopenIntervalChange tests changing the reopen interval
func TestFileReader_ReopenIntervalChange(t *testing.T) {
	reader := NewFileReader("test.log")

	// Default should be 1 second
	if reader.reopenInterval != 1*time.Second {
		t.Errorf("Expected default reopenInterval to be 1s, got %v", reader.reopenInterval)
	}

	// Change interval
	newInterval := 500 * time.Millisecond
	reader.reopenInterval = newInterval

	if reader.reopenInterval != newInterval {
		t.Errorf("Expected reopenInterval to be %v after change, got %v",
			newInterval, reader.reopenInterval)
	}
}
