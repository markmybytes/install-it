package porter

import (
	"context"
	"errors"
	"install-it/pkg/status"
	"sync"
	"time"
)

// JobSnapshot is a point-in-time view of the current job, polled by the frontend.
type JobSnapshot struct {
	Status   status.Status `json:"status"`   // pending|running|completed|failed|aborted
	Step     string        `json:"step"`     // task phase; consumers define their own list (see ProgressModal)
	Progress float64       `json:"progress"` // 0.0 to 1.0
	Messages []string      `json:"messages"` // recent messages (tail)
}

// job tracks the state of a single export/import operation.
type job struct {
	mu       sync.Mutex
	status   status.Status
	step     string
	progress float64
	startAt  time.Time

	ctx      context.Context
	cancel   context.CancelFunc
	messages chan string // buffered channel for receiving messages from worker
}

func newJob() *job {
	ctx, cancel := context.WithCancel(context.Background())
	return &job{
		status:   status.Pending,
		ctx:      ctx,
		cancel:   cancel,
		messages: make(chan string, 4096),
	}
}

func (j *job) start() {
	j.mu.Lock()
	j.status = status.Running
	j.startAt = time.Now()
	j.mu.Unlock()
}

func (j *job) setStep(name string) {
	j.mu.Lock()
	j.step = name
	j.mu.Unlock()
}

func (j *job) setProgress(p float64) {
	j.mu.Lock()
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	j.progress = p
	j.mu.Unlock()
}

// msg sends a message to the channel (non-blocking). Messages are dropped when the
// buffer is full — the consumer must poll snapshot() to drain.
func (j *job) msg(s string) {
	select {
	case j.messages <- s:
	default:
	}
}

func (j *job) complete() {
	j.mu.Lock()
	j.status = status.Completed
	j.progress = 1.0
	j.mu.Unlock()
}

// fail sets status=Failed (or Aborted if err is context.Canceled).
func (j *job) fail(err error) {
	j.mu.Lock()
	if errors.Is(err, context.Canceled) {
		j.status = status.Aborted
	} else {
		j.status = status.Failed
	}
	j.mu.Unlock()
}

// snapshot drains messages (destructive) and returns a point-in-time view of the job.
// Messages is the delta since the last snapshot call — the consumer accumulates.
//
// Single-poller assumption: the consumer calls snapshot() sequentially.
func (j *job) snapshot() JobSnapshot {
	j.mu.Lock()
	defer j.mu.Unlock()

	// If the context was cancelled, override the returned status
	// so the caller sees Aborting/Aborted even if complete() raced ahead.
	snapStatus := j.status
	if j.ctx.Err() == context.Canceled {
		switch j.status {
		case status.Running, status.Aborting:
			snapStatus = status.Aborting
		default:
			snapStatus = status.Aborted
		}
	}

	var msgs []string
	for {
		select {
		case m := <-j.messages:
			msgs = append(msgs, m)
		default:
			goto done
		}
	}
done:

	return JobSnapshot{
		Status:   snapStatus,
		Step:     j.step,
		Progress: j.progress,
		Messages: msgs,
	}
}
