//go:build !windows

package dev

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type processHandle struct {
	cmd *exec.Cmd
}

func startProcess(ctx context.Context, binary, dir string, env []string) (*processHandle, error) {
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &processHandle{cmd: cmd}, nil
}

func stopProcess(proc *processHandle) {
	if proc == nil || proc.cmd == nil || proc.cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(proc.cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = proc.cmd.Process.Signal(syscall.SIGTERM)
	}

	done := make(chan error, 1)
	go func() {
		done <- proc.cmd.Wait()
	}()

	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		if pgid > 0 {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = proc.cmd.Process.Kill()
		}
		<-done
	}
}
