package logger

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sqve/grove/internal/styles"
)

type Spinner struct {
	message atomic.Value
	done    chan struct{}
	once    sync.Once
	wg      sync.WaitGroup
}

func StartSpinner(message string) *Spinner {
	s := &Spinner{done: make(chan struct{})}
	s.message.Store(message)

	if isPlain() {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Info, "→"), message)
		s.once.Do(func() { close(s.done) })
		return s
	}

	s.wg.Add(1)
	go s.animate()
	return s
}

func (s *Spinner) animate() {
	defer s.wg.Done()
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-s.done:
			fmt.Fprint(os.Stderr, "\r\033[K")
			return
		case <-ticker.C:
			msg, _ := s.message.Load().(string)
			fmt.Fprintf(os.Stderr, "\r%s %s",
				styles.Render(&styles.Info, frames[i]),
				msg)
			i = (i + 1) % len(frames)
		}
	}
}

func (s *Spinner) Update(message string) {
	s.message.Store(message)
}

func (s *Spinner) Stop() {
	s.once.Do(func() { close(s.done) })
	s.wg.Wait()
}

func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	Success("%s", message)
}

func (s *Spinner) StopWithError(message string) {
	s.Stop()
	Error("%s", message)
}
