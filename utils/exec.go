package utils

import (
	"bytes"
	"errors"
	"os"
	"os/exec"

	"github.com/yodamad/heimdall/entity"
)

func ExecCmd(cmd string, gf entity.GitFolder) entity.CmdInfo {
	execContext := "-c"
	if GetMorningRoutine().ohmyzsh {
		execContext = "-lic"
	}
	Trace("Executing command '"+cmd+"' in "+gf.Path, true)
	// #nosec G204
	command := exec.Command(GetMorningRoutine().Shell, execContext, cmd)
	command.Env = os.Environ()
	command.Dir = gf.Path

	var out, errb bytes.Buffer
	command.Stdout = &out
	command.Stderr = &errb

	err := command.Run()

	code := 0
	if err != nil {
		// Non-zero exit or start/context error
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// Command executed but exited with non-zero status
			code = ee.ExitCode()
		} else {
			// Failed to start (e.g., zsh not found) or context canceled
			code = -1
		}
	} else if ps := command.ProcessState; ps != nil {
		// Successful run, get 0 (or actual code) from ProcessState
		code = ps.ExitCode()
	}
	return entity.CmdInfo{
		Cmd:      cmd,
		ExitCode: code,
	}
}
