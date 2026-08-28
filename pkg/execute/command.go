package execute

import (
	"bytes"
	"errors"
	"os/exec"
	"time"

	"install-it/pkg/errcode"

	"github.com/saintfish/chardet"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/net/html/charset"
)

type Command struct {
	cmd       *exec.Cmd
	startTime time.Time
	stdout    bytes.Buffer
	stderr    bytes.Buffer
	stopped   bool
}

func NewCommand(program string, options []string) *Command {
	wrapper := Command{cmd: exec.Command(program, options...)}
	wrapper.cmd.Stdout = &wrapper.stdout
	wrapper.cmd.Stderr = &wrapper.stderr
	return &wrapper
}

func (t *Command) Start() error {
	t.startTime = time.Now()
	return t.cmd.Start()
}

func (t *Command) Wait() error {
	return t.cmd.Wait()
}

func (t *Command) Run() error {
	t.startTime = time.Now()
	return t.cmd.Run()
}

func (t *Command) Stop() error {
	if t.cmd.Process == nil {
		panic("execute: called Stop before command started")
	}

	proc, err := process.NewProcess(int32(t.cmd.Process.Pid))
	if err != nil {
		return errcode.New("errExecuteAbortFailed")
	}

	if children, err := proc.Children(); err != nil {
		return errcode.New("errExecuteAbortFailed")
	} else {
		var errorChain error = nil
		for _, p := range children {
			if err = p.Kill(); err != nil {
				errorChain = errors.Join(errorChain, errcode.New("errExecuteAbortFailed"))
			}
		}

		if err := proc.Kill(); err != nil {
			errorChain = errors.Join(errorChain, errcode.New("errExecuteAbortFailed"))
		}

		t.stopped = errorChain == nil

		return errorChain
	}
}

func (t Command) Lapse() float32 {
	if t.startTime.Year() == 1 {
		return -1.0
	}
	return float32(time.Since(t.startTime).Milliseconds()) / 1000
}

func (t Command) DecodeStdout() string {
	if s, err := t.DecodeStdPipe(t.stdout); err != nil {
		return t.stdout.String()
	} else {
		return s
	}
}

func (t Command) DecodeStderr() string {
	if s, err := t.DecodeStdPipe(t.stderr); err != nil {
		return t.stderr.String()
	} else {
		return s
	}
}

func (t Command) DecodeStdPipe(buff bytes.Buffer) (string, error) {
	detector := chardet.NewTextDetector()
	if result, err := detector.DetectBest(buff.Bytes()); err == nil {
		if encoding, _ := charset.Lookup(result.Charset); encoding != nil {
			return encoding.NewDecoder().String(buff.String())
		} else {
			return "", errcode.New("errExecuteDecodeFailed")
		}
	} else {
		return "", errcode.New("errExecuteDecodeFailed")
	}
}
