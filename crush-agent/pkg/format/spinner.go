package format

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Spinner provides simple text-based progress indication without UI dependencies.
type Spinner struct {
	done   chan struct{}
	label  string
	stopCh chan struct{}
}

// NewSpinner creates a simple text-based spinner.
func NewSpinner(ctx context.Context, cancel context.CancelFunc, label string) *Spinner {
	return &Spinner{
		done:   make(chan struct{}, 1),
		label:  label,
		stopCh: make(chan struct{}),
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.stopCh:
				fmt.Fprint(os.Stderr, "\r\033[K")
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], s.label)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
}

// Stop ends the spinner animation.
func (s *Spinner) Stop() {
	close(s.stopCh)
	<-s.done
	fmt.Fprint(os.Stderr, "\r\033[K")
}
