package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"nrcc/internal/server"
	"nrcc/internal/service"
)

func Run(args []string, frontend fs.FS) error {
	if len(args) == 0 {
		return start(frontend)
	}

	switch args[0] {
	case "setup":
		return setup()
	case "start":
		return start(frontend)
	case "stop":
		return stop()
	case "doctor":
		return doctor()
	case "version":
		return version()
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		printHelp()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func start(frontend fs.FS) error {
	envSvc := service.NewEnvironmentService()
	dataDir, err := envSvc.DefaultDataDir()
	if err != nil {
		return err
	}
	if err := envSvc.StartPreflight(dataDir); err != nil {
		return err
	}

	authService, err := service.NewAuthService(dataDir)
	if err != nil {
		return err
	}
	defer authService.Close()

	configService := service.NewConfigService(dataDir)
	managedEnvService := service.NewManagedEnvService(dataDir)
	backupService := service.NewBackupService(dataDir)

	port := envOrDefault("NRCC_PORT", "3000")
	runtimePort, err := strconv.Atoi(envOrDefault("NRCC_NODE_RED_PORT", "1880"))
	if err != nil {
		return fmt.Errorf("invalid NRCC_NODE_RED_PORT: %w", err)
	}

	processManager := service.NewProcessManager(service.ProcessConfig{
		DataDir: dataDir,
		Port:    runtimePort,
	})
	if err := processManager.Start(); err != nil {
		return err
	}

	app := server.New(AppConfig{
		Port:       port,
		Frontend:   frontend,
		Runtime:    processManager,
		Auth:       authService,
		Config:     configService,
		ManagedEnv: managedEnvService,
		Backups:    backupService,
	})

	fmt.Printf("nrcc listening on http://127.0.0.1:%s\n", port)
	fmt.Printf("node-red runtime on http://127.0.0.1:%d\n", runtimePort)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- app.Start()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-serverErr:
		_ = processManager.Stop()
		return err
	case <-signals:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		return err
	}
	if err := processManager.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return nil
}

func setup() error {
	envSvc := service.NewEnvironmentService()
	dataDir, err := envSvc.DefaultDataDir()
	if err != nil {
		return err
	}

	return envSvc.Setup(dataDir, os.Stdin, os.Stdout)
}

func stop() error {
	return errors.New("stop command is not implemented yet")
}

func doctor() error {
	envSvc := service.NewEnvironmentService()
	dataDir, err := envSvc.DefaultDataDir()
	if err != nil {
		return err
	}

	report := envSvc.Diagnose(dataDir)
	fmt.Printf("OS: %s/%s\n", report.OS, report.Arch)
	fmt.Printf("Data dir: %s\n", report.DataDir)
	for _, check := range report.Checks {
		fmt.Printf("- [%s] %s: %s\n", check.Status, check.Name, check.Detail)
	}

	if !report.NodeInstalled || !report.NPMInstalled || !report.NodeRedReady {
		return errors.New("environment is not ready")
	}

	return nil
}

func version() error {
	fmt.Println("nrcc dev")
	return nil
}

func printHelp() {
	fmt.Println("nrcc <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  setup     Prepare the local environment")
	fmt.Println("  start     Start the local control center")
	fmt.Println("  stop      Stop the local control center")
	fmt.Println("  doctor    Check local prerequisites")
	fmt.Println("  version   Print version information")
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type AppConfig = server.Config
