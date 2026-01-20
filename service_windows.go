//go:build windows

package main

import (
	"Syncase-silent-app-main/config"
	"Syncase-silent-app-main/uploader"
	"Syncase-silent-app-main/watcher"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const svcName = "SyncaseSilentService"

type service struct{}

func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the main app in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- runApp(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				// Trigger app shutdown
				cancel()
				// Wait for app to finish
				select {
				case <-done:
				case <-time.After(30 * time.Second):
				}
				return false, 0
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			}
		case err := <-done:
			if err != nil {
				log.Printf("Service stopped with error: %v", err)
			}
			return false, 0
		}
	}
}

func runService() {
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatal(err)
	}

	if !isInteractive {
		// Run as Windows Service
		elog, err := eventlog.Open(svcName)
		if err == nil {
			elog.Info(1, fmt.Sprintf("Starting %s service", svcName))
			defer elog.Close()
		}

		err = svc.Run(svcName, &service{})
		if err != nil {
			if elog != nil {
				elog.Error(1, fmt.Sprintf("Service failed: %v", err))
			}
			log.Printf("Service failed: %v", err)
		}
		return
	}

	// Run in console mode for debugging
	fmt.Println("Running in interactive mode (not as service)")
	if err := runApp(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Helper functions for service management
func installService() error {
	exePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", svcName)
	}

	s, err = m.CreateService(
		svcName,
		exePath,
		mgr.Config{
			DisplayName:  "Syncase Silent File Sync",
			Description:  "Syncs files to cloud storage silently in the background",
			StartType:    mgr.StartAutomatic,
			ErrorControl: mgr.ErrorNormal,
		},
		"run",
	)
	if err != nil {
		return err
	}
	defer s.Close()

	fmt.Printf("Service %s installed successfully\n", svcName)
	return nil
}

func uninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", svcName)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	fmt.Printf("Service %s uninstalled successfully\n", svcName)
	return nil
}

func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return err
	}

	fmt.Printf("Service %s started successfully\n", svcName)
	return nil
}

func stopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		return err
	}

	fmt.Printf("Service %s stopped successfully\n", svcName)
	return nil
}

func runApp(ctx context.Context) error {
	fmt.Println("🚀 Starting Syncase...")

	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	logFile, err := os.OpenFile("logs/sync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println("[INFO] Syncase started")

	cfg, err := config.LoadConfigFromFile("config.json")
	if err != nil {
		return err
	}

	cfg.WatchedFolder, err = filepath.Abs(cfg.WatchedFolder)
	if err != nil {
		return err
	}

	info, err := os.Stat(cfg.WatchedFolder)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("invalid watched folder: %s", cfg.WatchedFolder)
	}

	// Initial sync
	if err := uploader.SyncRemoteToLocal(ctx, cfg); err != nil {
		log.Println("[WARN] initial remote pull failed:", err)
	}

	// BLOCKS here (this is correct)
	return watcher.StartWatcher(ctx, cfg)
}
