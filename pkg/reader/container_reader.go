package reader

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ContainerReader represents a component that tails container logs
type ContainerReader struct {
	namespace     string
	podName       string
	containerName string
	clientset     *kubernetes.Clientset
	lines         chan string
	stopCh        chan struct{}
	stoppedCh     chan struct{}
	lock          sync.Mutex
	isRunning     bool
}

// NewContainerReaderFunc is the function type for creating container readers
type NewContainerReaderFunc func(namespace, podName, containerName string) (LogReader, error)

// NewContainerReader creates a new container log reader
var NewContainerReader = func(namespace, podName, containerName string) (LogReader, error) {
	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating in-cluster config: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes client: %v", err)
	}

	return &ContainerReader{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		clientset:     clientset,
		lines:         make(chan string, 1000),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
		isRunning:     false,
	}, nil
}

// Start begins the container log tailing process
func (r *ContainerReader) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.isRunning {
		return fmt.Errorf("container reader already started")
	}

	go r.tailContainer()
	r.isRunning = true
	return nil
}

// Lines returns the channel of log lines
func (r *ContainerReader) Lines() <-chan string {
	return r.lines
}

// Stop stops the container reader
func (r *ContainerReader) Stop() {
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

// tailContainer continuously reads the container logs and sends them to the channel
func (r *ContainerReader) tailContainer() {
	defer close(r.stoppedCh)

	for {
		select {
		case <-r.stopCh:
			return
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
			defer cancel()

			// Create pod logs request
			req := r.clientset.CoreV1().Pods(r.namespace).GetLogs(r.podName, &corev1.PodLogOptions{
				Container: r.containerName,
				Follow:    true,
				TailLines: int64Ptr(10), // Start with the last 10 lines
			})

			// Get stream of logs
			stream, err := req.Stream(ctx)
			if err != nil {
				fmt.Printf("Error opening stream: %v\n", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Read logs line by line
			reader := NewLogLineReader(stream)
			for {
				line, err := reader.ReadLine()
				if err != nil {
					if err != io.EOF {
						fmt.Printf("Error reading log line: %v\n", err)
					}
					break
				}

				select {
				case r.lines <- line:
					// Line sent successfully
				case <-r.stopCh:
					stream.Close()
					return
				}
			}

			stream.Close()

			// Check if pod still exists
			_, err = r.clientset.CoreV1().Pods(r.namespace).Get(ctx, r.podName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("Pod %s/%s no longer exists: %v\n", r.namespace, r.podName, err)
				return
			}

			// Wait a bit before reconnecting
			time.Sleep(5 * time.Second)
		}
	}
}

// LogLineReader reads lines from a log stream
type LogLineReader struct {
	reader io.Reader
	buffer []byte
}

// NewLogLineReader creates a new log line reader
func NewLogLineReader(reader io.Reader) *LogLineReader {
	return &LogLineReader{
		reader: reader,
		buffer: make([]byte, 0, 4096),
	}
}

// ReadLine reads a line from the log stream
func (r *LogLineReader) ReadLine() (string, error) {
	// Read data in chunks
	chunk := make([]byte, 1024)
	for {
		// Find newline in existing buffer
		for i, b := range r.buffer {
			if b == '\n' {
				line := string(r.buffer[:i])
				r.buffer = r.buffer[i+1:]
				return line, nil
			}
		}

		// Read more data
		n, err := r.reader.Read(chunk)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 && err == io.EOF {
			// End of file, return any remaining data as a line
			if len(r.buffer) > 0 {
				line := string(r.buffer)
				r.buffer = r.buffer[:0]
				return line, nil
			}
			return "", io.EOF
		}

		// Append chunk to buffer
		r.buffer = append(r.buffer, chunk[:n]...)
	}
}

// int64Ptr converts an int64 to a pointer
func int64Ptr(i int64) *int64 {
	return &i
}
