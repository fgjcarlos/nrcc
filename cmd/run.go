package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"nrcc/internal/security"
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
	case "doctor":
		return runDoctor(args[1:])
	case "logs":
		return runLogs(args[1:])
	case "support":
		return runSupport(args[1:])
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
	// Set up structured logging
	logLevel := slog.LevelInfo
	if os.Getenv("NRCC_LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

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

	// Initialize schemas
	if err := service.InitLogSchema(authService.GetDB()); err != nil {
		return err
	}
	if err := service.InitConfigSnapshotSchema(authService.GetDB()); err != nil {
		return err
	}

	logService, err := service.NewLogService(dataDir, authService.GetDB())
	if err != nil {
		return err
	}
	defer logService.Close()

	jobsService := service.NewJobsService(authService.GetDB())

	configService := service.NewConfigService(dataDir, authService.GetDB())
	managedEnvService := service.NewManagedEnvService(dataDir)
	backupService := service.NewBackupService(dataDir)
	libraryService := service.NewLibraryService(dataDir)
	flowService := service.NewFlowService(dataDir)
	updateService := service.NewUpdateService(dataDir, &backupService)
	operationLock := service.NewOperationLock()

	// Create security, doctor, and support bundle services
	sanitizer := security.NewSanitizer()
	doctorService := service.NewDoctorService(dataDir)
	supportBundleService := service.NewSupportBundleService(dataDir, logService, doctorService, sanitizer)

	port := envOrDefault("NRCC_PORT", "3000")
	localAccessPort, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid NRCC_PORT: %w", err)
	}
	runtimePort, err := strconv.Atoi(envOrDefault("NRCC_NODE_RED_PORT", "1880"))
	if err != nil {
		return fmt.Errorf("invalid NRCC_NODE_RED_PORT: %w", err)
	}
	localAccessService := service.NewLocalAccessService(localAccessPort)
	assetService := service.NewAssetService(dataDir)

	processManager := service.NewProcessManager(service.ProcessConfig{
		DataDir: dataDir,
		Port:    runtimePort,
	})

	// Wire up LogService and JobsService to all services
	processManager.SetLogService(logService)
	operationLock.SetLogService(logService)
	backupService.SetLogService(logService)
	backupService.SetJobsService(jobsService)
	updateService.SetLogService(logService)
	updateService.SetJobsService(jobsService)

	// Wire up services to DoctorService
	doctorService.SetProcessManager(processManager)
	doctorService.SetLogService(logService)
	doctorService.SetLocalAccessService(localAccessService)

	if err := processManager.Start(); err != nil {
		return err
	}

	app := server.New(AppConfig{
		Port:        port,
		Frontend:    frontend,
		Runtime:     processManager,
		Auth:        authService,
		Config:      configService,
		ManagedEnv:  managedEnvService,
		Backups:     backupService,
		Libraries:   libraryService,
		Flows:       flowService,
		Updates:     updateService,
		Operations:  operationLock,
		Logs:        logService,
		Jobs:        jobsService,
		Doctor:      doctorService,
		Support:     supportBundleService,
		LocalAccess: localAccessService,
		Assets:      &assetService,
	})

	localAccess := localAccessService.EnsureConfigured()

	slog.Info("nrcc listening", "addr", "http://127.0.0.1:"+port)
	slog.Info("preferred local access", "url", localAccess.URL)
	if localAccess.URL != localAccess.FallbackURL {
		slog.Info("fallback local access", "url", localAccess.FallbackURL)
	}
	if localAccess.Message != "" {
		slog.Info("local access status", "message", localAccess.Message)
	}
	slog.Info("node-red runtime", "addr", fmt.Sprintf("http://127.0.0.1:%d", runtimePort))

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
	fmt.Println("  doctor    Check local prerequisites")
	fmt.Println("  logs      View system logs")
	fmt.Println("  support   Generate support bundle for diagnostics")
	fmt.Println("  version   Print version information")
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type AppConfig = server.Config
