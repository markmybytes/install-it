package execute

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os/exec"
	"syscall"

	"install-it/pkg/errcode"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type CommandExecutor struct {
	ctx      context.Context
	commands *xsync.MapOf[string, *Command]
}

type CommandResult struct {
	Lapse    float32 `json:"lapse"`
	ExitCode int     `json:"exitCode"`
	Stdout   string  `json:"stdout"`
	Stderr   string  `json:"stderr"`
	Error    string  `json:"error"`
	Aborted  bool    `json:"aborted"`
}

func (ce *CommandExecutor) SetContext(ctx context.Context) {
	ce.ctx = ctx
	ce.commands = xsync.NewMapOf[string, *Command]()
}

func (ce *CommandExecutor) Run(program string, options []string) string {
	id := ce.generateId()
	ce.commands.Store(id, NewCommand(program, options))

	go ce.dispatch(id)

	return id
}

func (ce *CommandExecutor) RunAndOutput(program string, options []string, hideWindow bool) CommandResult {
	var (
		errMsg  string
		command = NewCommand(program, options)
	)

	if hideWindow {
		command.cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
	}

	if err := command.Run(); err != nil {
		errMsg = runFailureCode(err)
	}

	return CommandResult{
		command.Lapse(),
		command.cmd.ProcessState.ExitCode(),
		command.stdout.String(),
		command.stderr.String(),
		errMsg,
		command.stopped,
	}
}

func (ce *CommandExecutor) Abort(id string) error {
	if task, ok := ce.commands.Load(id); !ok {
		return errcode.New("errExecuteIdNotFound")
	} else {
		if err := task.Stop(); err != nil {
			return errcode.New("errExecuteAbortFailed")
		}
		return nil
	}
}

func (ce *CommandExecutor) dispatch(id string) {
	command, ok := ce.commands.Load(id)
	if !ok {
		runtime.EventsEmit(ce.ctx, "execute:exited", id, CommandResult{
			Error: "errExecuteIdNotFound",
		})
		return
	}

	var errMsg string
	if err := command.Run(); err != nil {
		errMsg = runFailureCode(err)
	}

	runtime.EventsEmit(ce.ctx, "execute:exited", id, CommandResult{
		command.Lapse(),
		command.cmd.ProcessState.ExitCode(),
		command.DecodeStdout(),
		command.DecodeStderr(),
		errMsg,
		command.stopped,
	})
}

func (ce CommandExecutor) generateId() string {
	id := ""
	for id == "" {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			continue
		}

		tmpId := hex.EncodeToString(b)
		if _, ok := ce.commands.Load(tmpId); ok {
			continue
		}

		id = tmpId
	}
	return id
}

// runFailureCode maps a `command.Run()` error to an errcode code string.
// *exec.ExitError is not a startup failure (the process exited non-zero),
// so it returns "" and the frontend uses the exit code + stderr for display.
// Any other error (path not found, permission denied, …) returns a single
// code; the raw error text is already in CommandResult.Stderr.
func runFailureCode(err error) string {
	if err == nil {
		return ""
	}
	if _, ok := err.(*exec.ExitError); ok {
		return ""
	}
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) ||
		errors.Is(err, syscall.ERROR_PATH_NOT_FOUND) {
		return "errFileNotFound"
	}
	return "errExecuteRunFailed"
}
