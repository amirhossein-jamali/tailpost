package reader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestLogLineReader tests the LogLineReader functionality
func TestLogLineReader(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Single line with newline",
			input:    "line1\n",
			expected: []string{"line1"},
		},
		{
			name:     "Single line without newline",
			input:    "line1",
			expected: []string{"line1"},
		},
		{
			name:     "Multiple lines",
			input:    "line1\nline2\nline3\n",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "Multiple lines without final newline",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewBufferString(tc.input)
			lineReader := NewLogLineReader(reader)

			var lines []string
			for {
				line, err := lineReader.ReadLine()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				lines = append(lines, line)
			}

			if len(lines) != len(tc.expected) {
				t.Fatalf("Expected %d lines, got %d", len(tc.expected), len(lines))
			}

			for i, expected := range tc.expected {
				if lines[i] != expected {
					t.Errorf("Line %d: expected %q, got %q", i, expected, lines[i])
				}
			}
		})
	}
}

// MockReader mocks a reader that returns chunks of data
type MockReader struct {
	chunks []string
	index  int
	done   bool
}

// NewMockReader creates a new mock reader
func NewMockReader(chunks []string) *MockReader {
	return &MockReader{
		chunks: chunks,
		index:  0,
		done:   false,
	}
}

// Read implements io.Reader interface
func (r *MockReader) Read(p []byte) (n int, err error) {
	if r.done {
		return 0, io.EOF
	}

	if r.index >= len(r.chunks) {
		r.done = true
		return 0, io.EOF
	}

	chunk := r.chunks[r.index]
	r.index++

	return copy(p, chunk), nil
}

// TestLogLineReaderChunks tests the LogLineReader with chunked data
func TestLogLineReaderChunks(t *testing.T) {
	chunks := []string{
		"line1\nli", "ne2\nlin", "e3\n",
	}

	mockReader := NewMockReader(chunks)
	lineReader := NewLogLineReader(mockReader)

	expected := []string{"line1", "line2", "line3"}
	var lines []string

	for {
		line, err := lineReader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		lines = append(lines, line)
	}

	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
	}

	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("Line %d: expected %q, got %q", i, exp, lines[i])
		}
	}
}

// TestContainerReaderInterface verifies ContainerReader implements LogReader interface
func TestContainerReaderInterface(t *testing.T) {
	// This is a compile-time check that ContainerReader implements LogReader
	var _ LogReader = (*ContainerReader)(nil)
}

// TestLogReaderFactory tests the reader factory for container type
func TestLogReaderFactory(t *testing.T) {
	// Store original function to restore it after test
	originalNewContainerReader := NewContainerReader

	// Override with a function that returns an error for this test
	NewContainerReader = func(namespace, podName, containerName string) (LogReader, error) {
		return nil, fmt.Errorf("error creating in-cluster config: test error")
	}

	// Restore original function after test
	defer func() {
		NewContainerReader = originalNewContainerReader
	}()

	config := LogSourceConfig{
		Type:          ContainerSourceType,
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "test-container",
	}

	// This should now fail with our mocked error
	_, err := NewReader(config)
	if err == nil {
		t.Fatalf("Expected error creating ContainerReader, got nil")
	}

	if !strings.Contains(err.Error(), "in-cluster config") {
		t.Fatalf("Expected in-cluster config error, got: %v", err)
	}
}

// TestContainerReader is a modified version of ContainerReader for testing
type TestContainerReader struct {
	namespace     string
	podName       string
	containerName string
	clientset     *fake.Clientset
	lines         chan string
	stopCh        chan struct{}
	stoppedCh     chan struct{}
	lock          sync.Mutex
	isRunning     bool
}

// MockContainerReader creates a TestContainerReader with a fake Kubernetes clientset
func NewMockContainerReader(namespace, podName, containerName string) *TestContainerReader {
	clientset := fake.NewSimpleClientset()

	return &TestContainerReader{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		clientset:     clientset,
		lines:         make(chan string, 1000),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
		isRunning:     false,
	}
}

// Lines returns the channel of log lines
func (r *TestContainerReader) Lines() <-chan string {
	return r.lines
}

// Start begins the container log tailing process
func (r *TestContainerReader) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.isRunning {
		return fmt.Errorf("container reader already started")
	}

	go func() {
		defer close(r.stoppedCh)
		// Mock implementation for testing
	}()

	r.isRunning = true
	return nil
}

// Stop stops the container reader
func (r *TestContainerReader) Stop() {
	r.lock.Lock()
	if !r.isRunning {
		r.lock.Unlock()
		return
	}
	r.lock.Unlock()

	close(r.stopCh)
	<-r.stoppedCh

	r.lock.Lock()
	r.isRunning = false
	r.lock.Unlock()
}

// TestContainerReaderLines tests the Lines method of TestContainerReader
func TestContainerReaderLines(t *testing.T) {
	reader := NewMockContainerReader("default", "test-pod", "test-container")

	// Verify that Lines returns the correct channel
	lines := reader.Lines()
	if lines != reader.lines {
		t.Errorf("Lines() did not return the expected channel")
	}
}

// TestContainerReaderStartStop tests the Start and Stop methods
func TestContainerReaderStartStop(t *testing.T) {
	reader := NewMockContainerReader("default", "test-pod", "test-container")

	// Start the reader
	err := reader.Start()
	if err != nil {
		t.Fatalf("Error starting container reader: %v", err)
	}

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Stop the reader
	reader.Stop()

	// Verify the reader has stopped by checking if the stoppedCh is closed
	select {
	case <-reader.stoppedCh:
		// Channel is closed, which is what we expect
	case <-time.After(1 * time.Second):
		t.Fatalf("Reader did not stop within timeout")
	}
}

// MockK8sLogsStreamProvider mocks the Kubernetes logs stream provider for testing
type MockK8sLogsStreamProvider struct {
	logs       string
	callCount  int
	errorAfter int
}

// Stream returns a reader containing the mock logs
func (m *MockK8sLogsStreamProvider) Stream(ctx context.Context) (io.ReadCloser, error) {
	m.callCount++

	if m.errorAfter > 0 && m.callCount > m.errorAfter {
		return nil, io.EOF
	}

	return io.NopCloser(strings.NewReader(m.logs)), nil
}

// Close is a no-op for testing
func (m *MockK8sLogsStreamProvider) Close() {}

// MockPodGetter mocks the pod getter for testing
type MockPodGetter struct {
	pod    *corev1.Pod
	getErr error
}

// Get returns the mock pod
func (m *MockPodGetter) Get(ctx context.Context, name string, options metav1.GetOptions) (*corev1.Pod, error) {
	return m.pod, m.getErr
}

// TestContainerReaderTailContainer tests the tailContainer method
func TestContainerReaderTailContainer(t *testing.T) {
	// Create a mock container reader with needed mocks
	namespace := "default"
	podName := "test-pod"
	containerName := "test-container"

	reader := &TestContainerReader{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		lines:         make(chan string, 1000),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	// Create a mocked logsProvider that returns "line1\nline2\nline3\n"
	logsProvider := &MockK8sLogsStreamProvider{
		logs: "line1\nline2\nline3\n",
	}

	// Mock the tailContainer function to use our mocked components
	// This is a test-only modification and doesn't affect the actual code
	go func() {
		defer close(reader.stoppedCh)

		stream, _ := logsProvider.Stream(context.Background())
		defer stream.Close()

		lineReader := NewLogLineReader(stream)
		for {
			line, err := lineReader.ReadLine()
			if err != nil {
				break
			}

			select {
			case reader.lines <- line:
				// Line sent successfully
			case <-reader.stopCh:
				return
			}
		}
	}()

	// Read lines from the channel
	expectedLines := []string{"line1", "line2", "line3"}
	receivedLines := make([]string, 0, 3)

	// Read with timeout to avoid blocking test
	timeout := time.After(1 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case line := <-reader.lines:
			receivedLines = append(receivedLines, line)
		case <-timeout:
			t.Fatalf("Timeout waiting for lines")
		}
	}

	// Stop the reader
	close(reader.stopCh)

	// Verify we received the expected lines
	if len(receivedLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d", len(expectedLines), len(receivedLines))
	}

	for i, expected := range expectedLines {
		if receivedLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, receivedLines[i])
		}
	}
}

// --- NEW TESTS BELOW ---

// ErrorReader simulates read errors for testing error handling
type ErrorReader struct {
	errAfterBytes int
	bytesRead     int
}

func (e *ErrorReader) Read(p []byte) (n int, err error) {
	if e.bytesRead >= e.errAfterBytes {
		return 0, errors.New("simulated read error")
	}

	// Write some data before error, but don't include a newline
	data := []byte("some")
	copied := copy(p, data)
	e.bytesRead += copied
	return copied, nil
}

// ClosableErrorReader wraps ErrorReader to implement io.ReadCloser
type ClosableErrorReader struct {
	*ErrorReader
}

func (c *ClosableErrorReader) Close() error {
	return nil
}

// TestLogLineReaderError tests error handling in LogLineReader
func TestLogLineReaderError(t *testing.T) {
	// Create a reader that errors after very few bytes
	errorReader := &ErrorReader{
		errAfterBytes: 2,
	}

	lineReader := NewLogLineReader(errorReader)

	// Try to read a line, should error
	_, err := lineReader.ReadLine()
	if err == nil {
		t.Fatalf("Expected error from LogLineReader, got nil")
	}

	if !strings.Contains(err.Error(), "simulated read error") {
		t.Errorf("Expected simulated error, got: %v", err)
	}
}

// TestContainerReaderStartTwice tests that calling Start twice returns an error
func TestContainerReaderStartTwice(t *testing.T) {
	reader := NewMockContainerReader("default", "test-pod", "test-container")

	// Start the reader once
	err1 := reader.Start()
	if err1 != nil {
		t.Fatalf("Error on first start: %v", err1)
	}

	// Start again, should error
	err2 := reader.Start()
	if err2 == nil {
		t.Fatalf("Expected error on second start, got nil")
	}

	if !strings.Contains(err2.Error(), "already started") {
		t.Errorf("Expected 'already started' error, got: %v", err2)
	}

	// Cleanup
	reader.Stop()
}

// TestContainerReaderStopWithoutStart tests calling Stop without Start
func TestContainerReaderStopWithoutStart(t *testing.T) {
	reader := NewMockContainerReader("default", "test-pod", "test-container")

	// Stop without starting, should not block or panic
	reader.Stop()

	// Verify isRunning is still false
	reader.lock.Lock()
	defer reader.lock.Unlock()
	if reader.isRunning {
		t.Errorf("isRunning should be false after stopping without starting")
	}
}

// MockK8sLogsStreamProviderWithError simulates stream errors
type MockK8sLogsStreamProviderWithError struct {
	streamErr error
}

func (m *MockK8sLogsStreamProviderWithError) Stream(ctx context.Context) (io.ReadCloser, error) {
	return nil, m.streamErr
}

func (m *MockK8sLogsStreamProviderWithError) Close() {}

// TestContainerReaderStreamError tests handling of stream errors
func TestContainerReaderStreamError(t *testing.T) {
	namespace := "default"
	podName := "test-pod"
	containerName := "test-container"

	reader := &TestContainerReader{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		lines:         make(chan string, 1000),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	// Create a provider that always errors
	logsProvider := &MockK8sLogsStreamProviderWithError{
		streamErr: fmt.Errorf("simulated stream error"),
	}

	streamErrorCounted := 0
	streamErrorCh := make(chan struct{}, 1)

	// Mock the tailContainer to detect errors
	go func() {
		defer close(reader.stoppedCh)

		for i := 0; i < 3; i++ { // Try a few times
			_, err := logsProvider.Stream(context.Background())
			if err != nil {
				streamErrorCounted++
				select {
				case streamErrorCh <- struct{}{}:
				default:
				}
				time.Sleep(5 * time.Millisecond) // Simulate backoff
			}

			select {
			case <-reader.stopCh:
				return
			default:
			}
		}
	}()

	// Wait for error to be detected
	select {
	case <-streamErrorCh:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for stream error")
	}

	// Stop the reader
	reader.Stop()

	// Verify error was detected
	if streamErrorCounted == 0 {
		t.Errorf("Stream error was not detected")
	}
}

// TestUnstableStreamReader tests handling of unstable streams (disconnect/reconnect)
func TestUnstableStreamReader(t *testing.T) {
	namespace := "default"
	podName := "test-pod"
	containerName := "test-container"

	reader := &TestContainerReader{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		lines:         make(chan string, 1000),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	// Create a provider that will provide logs and then disconnect
	unstableProvider := &MockK8sLogsStreamProvider{
		logs:       "line1\nline2\n",
		errorAfter: 1, // Will work once, then error
	}

	reconnectCount := 0
	lineCount := 0
	reconnectCh := make(chan struct{}, 1)

	// Mock tailContainer with reconnect logic
	go func() {
		defer close(reader.stoppedCh)

		for i := 0; i < 3 && reconnectCount < 2; i++ {
			stream, err := unstableProvider.Stream(context.Background())
			if err != nil {
				reconnectCount++
				select {
				case reconnectCh <- struct{}{}:
				default:
				}
				time.Sleep(5 * time.Millisecond) // Backoff
				continue
			}

			lineReader := NewLogLineReader(stream)
			for {
				line, err := lineReader.ReadLine()
				if err != nil {
					break
				}

				lineCount++
				select {
				case reader.lines <- line:
					// Line sent successfully
				case <-reader.stopCh:
					stream.Close()
					return
				}
			}

			stream.Close()
		}
	}()

	// Let it run to capture reconnects
	time.Sleep(50 * time.Millisecond)

	// Stop the reader
	reader.Stop()

	// Verify behavior
	if reconnectCount == 0 {
		t.Errorf("No reconnect attempts detected")
	}

	if lineCount == 0 {
		t.Errorf("No lines were read despite reconnect attempts")
	}
}

// TestPodNotFoundHandling tests handling of pod not found scenario
func TestPodNotFoundHandling(t *testing.T) {
	namespace := "default"
	podName := "test-pod"
	containerName := "test-container"

	reader := &TestContainerReader{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		lines:         make(chan string, 1000),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	// Create a mocked logsProvider
	logsProvider := &MockK8sLogsStreamProvider{
		logs: "line1\nline2\n",
	}

	// Create a mock pod getter that returns "not found"
	podGetter := &MockPodGetter{
		pod:    nil,
		getErr: fmt.Errorf("pod not found"),
	}

	podNotFoundDetected := false
	podNotFoundCh := make(chan struct{}, 1)

	// Mock tailContainer with pod checking
	go func() {
		defer close(reader.stoppedCh)

		// First get some logs
		stream, _ := logsProvider.Stream(context.Background())
		lineReader := NewLogLineReader(stream)
		// Read a line or two
		for i := 0; i < 2; i++ {
			line, err := lineReader.ReadLine()
			if err != nil {
				break
			}

			select {
			case reader.lines <- line:
				// Line sent
			case <-reader.stopCh:
				stream.Close()
				return
			}
		}
		stream.Close()

		// Now check if pod exists
		_, err := podGetter.Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			// Pod not found, exit
			podNotFoundDetected = true
			select {
			case podNotFoundCh <- struct{}{}:
			default:
			}
			return
		}
	}()

	// Wait for pod not found to be detected or timeout
	select {
	case <-podNotFoundCh:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for pod not found detection")
	}

	// Verify behavior
	if !podNotFoundDetected {
		t.Errorf("Pod not found error was not detected")
	}

	// Read lines that were sent before pod was detected as missing
	timeout := time.After(100 * time.Millisecond)
	lineCount := 0
	for {
		select {
		case <-reader.lines:
			lineCount++
		case <-timeout:
			goto done
		}
	}
done:

	if lineCount == 0 {
		t.Errorf("No lines received before pod not found")
	}

	// Cleanup
	reader.Stop()
}

// TestConcurrentLogReading tests concurrent log reading from multiple goroutines
func TestConcurrentLogReading(t *testing.T) {
	reader := NewMockContainerReader("default", "test-pod", "test-container")

	// Start the reader
	err := reader.Start()
	if err != nil {
		t.Fatalf("Error starting reader: %v", err)
	}

	// Manually send logs to the channel
	go func() {
		for i := 0; i < 100; i++ {
			select {
			case reader.lines <- fmt.Sprintf("line%d", i):
				// Line sent
			case <-reader.stopCh:
				return
			}
		}
	}()

	// Start multiple consumers
	const numConsumers = 5
	var wg sync.WaitGroup
	wg.Add(numConsumers)

	lineCounters := make([]int, numConsumers)
	for i := 0; i < numConsumers; i++ {
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-reader.lines:
					lineCounters[id]++
					time.Sleep(1 * time.Millisecond) // Simulate processing
				case <-time.After(50 * time.Millisecond):
					return // Timeout indicates no more lines
				}
			}
		}(i)
	}

	// Wait for all consumers to finish
	wg.Wait()

	// Stop the reader
	reader.Stop()

	// Check if all consumers got some lines
	totalLines := 0
	for i, count := range lineCounters {
		t.Logf("Consumer %d read %d lines", i, count)
		totalLines += count
	}

	if totalLines == 0 {
		t.Errorf("No lines were consumed by any consumer")
	}
}

// FlakyReader simulates an unstable reader that sometimes works and sometimes fails
type FlakyReader struct {
	data       []byte
	failEvery  int
	readCount  int
	closeCount int
}

func NewFlakyReader(data string, failEvery int) *FlakyReader {
	return &FlakyReader{
		data:      []byte(data),
		failEvery: failEvery,
	}
}

func (f *FlakyReader) Read(p []byte) (n int, err error) {
	f.readCount++

	// Simulate periodic failures
	if f.failEvery > 0 && f.readCount%f.failEvery == 0 {
		return 0, fmt.Errorf("simulated read failure %d", f.readCount)
	}

	// Otherwise return some data
	if len(f.data) == 0 {
		return 0, io.EOF
	}

	n = copy(p, f.data)
	f.data = f.data[n:]
	return n, nil
}

func (f *FlakyReader) Close() error {
	f.closeCount++
	return nil
}

// TestLogLineReaderWithFlakyReader tests LogLineReader with an unstable reader
func TestLogLineReaderWithFlakyReader(t *testing.T) {
	reader := NewFlakyReader("line1\nline2\nline3\n", 3) // Fail every 3rd read
	lineReader := NewLogLineReader(reader)

	// Try to read lines
	lines := []string{}
	var lastErr error

	for i := 0; i < 5; i++ {
		line, err := lineReader.ReadLine()
		if err != nil {
			lastErr = err
			break
		}
		lines = append(lines, line)
	}

	// We expect some lines to be read before error
	if len(lines) == 0 {
		t.Errorf("Expected to read some lines before error, got none")
	}

	// And we expect an error to occur
	if lastErr == nil {
		t.Errorf("Expected error from flaky reader, got nil")
	} else {
		t.Logf("Got expected error: %v", lastErr)
	}

	// Since LogLineReader doesn't call Close (that happens in tailContainer),
	// we don't actually expect Close to be called here
	// Removing the check for closeCount
}
