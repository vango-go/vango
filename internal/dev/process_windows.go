//go:build windows

package dev

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processHandle struct {
	cmd *exec.Cmd
	job windows.Handle
}

func startProcess(ctx context.Context, binary, dir string, env []string) (*processHandle, error) {
	job, err := createJobObject()
	if err != nil {
		job = 0
	}

	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		if job != 0 {
			windows.CloseHandle(job)
		}
		return nil, err
	}

	if job != 0 {
		if err := assignProcessToJob(job, cmd.Process.Pid); err != nil {
			windows.CloseHandle(job)
			job = 0
		}
	}

	return &processHandle{cmd: cmd, job: job}, nil
}

func stopProcess(proc *processHandle) {
	if proc == nil || proc.cmd == nil || proc.cmd.Process == nil {
		return
	}

	if proc.job != 0 {
		windows.CloseHandle(proc.job)
		proc.job = 0
	} else {
		_ = proc.cmd.Process.Kill()
	}

	done := make(chan error, 1)
	go func() {
		done <- proc.cmd.Wait()
	}()

	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		_ = proc.cmd.Process.Kill()
		<-done
	}
}

func createJobObject() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		windows.CloseHandle(job)
		return 0, err
	}

	return job, nil
}

func assignProcessToJob(job windows.Handle, pid int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	return windows.AssignProcessToJobObject(job, handle)
}
