// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

// Implements support for running the storagenode-updater as a Windows Service.
//
// The Windows Service can be created with sc.exe, e.g.
//
// sc.exe create storagenode-updater binpath= "C:\Users\MyUser\storagenode-updater.exe run ..."

// +build windows,!unittest

package main

import (
	"log"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/windows/svc"

	"storj.io/storj/pkg/process"
)

func init() {
	// Check if session is interactive
	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		panic("Failed to determine if session is interactive:" + err.Error())
	}

	if interactive {
		return
	}

	// Check if the 'run' command is invoked
	if len(os.Args) < 2 {
		return
	}

	if os.Args[1] != "run" {
		return
	}

	// Initialize the Windows Service handler
	err = svc.Run("storagenode-updater", &service{})
	if err != nil {
		panic("Service failed: " + err.Error())
	}
	// avoid starting main() when service was stopped
	os.Exit(0)
}

type service struct{}

func (m *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	var group errgroup.Group
	group.Go(func() error {
		process.Exec(rootCmd)
		return nil
	})

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			log.Println("Interrogate request received.")
			changes <- c.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			log.Println("Stop/Shutdown request received.")
			changes <- svc.Status{State: svc.StopPending}
			// Cancel the command's root context to cleanup resources
			_, cancel := process.Ctx(runCmd)
			cancel()
			_ = group.Wait() // process.Exec does not return an error
			// After returning the Windows Service is stopped and the process terminates
			return
		default:
			log.Println("Unexpected control request:", c)
		}
	}
	return
}