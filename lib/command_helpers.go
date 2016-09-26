package lib

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"time"
)

func runCommand(cmd *exec.Cmd) (string, error) {
	out := &bytes.Buffer{}
	buildOutput := func() string { return strings.TrimSpace(out.String()) }
	cmd.Stderr = out
	cmd.Stdout = out
	err := cmd.Start()
	if err != nil {
		return buildOutput(), err
	}
	cmdErrChan := make(chan error, 1)
	go func() { cmdErrChan <- cmd.Wait() }()
	select {
	case <-time.After(30 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return buildOutput(), err
		}
	case err := <-cmdErrChan:
		if err != nil {
			return buildOutput(), err
		}
	}
	return buildOutput(), nil
}

func runCommandWithStdin(in io.Reader, cmd *exec.Cmd) (string, error) {
	cmd.Stdin = in
	return runCommand(cmd)
}
