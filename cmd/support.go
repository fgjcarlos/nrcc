package cmd

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"nrcc/internal/db"
	"nrcc/internal/security"
	"nrcc/internal/service"
)

// runSupport handles the "nrcc support" command
func runSupport(args []string) error {
	fs := flag.NewFlagSet("support", flag.ContinueOnError)
	openFlag := fs.Bool("open", false, "open the support directory in file manager")
	fs.Parse(args)

	envSvc := service.NewEnvironmentService()
	dataDir, err := envSvc.DefaultDataDir()
	if err != nil {
		return err
	}

	// Open database with migrations
	dbPath := filepath.Join(dataDir, "nrcc.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	logService, err := service.NewLogService(dataDir, database)
	if err != nil {
		return err
	}
	defer logService.Close()

	sanitizer := security.NewSanitizer()
	doctorService := service.NewDoctorService(dataDir)
	supportBundleService := service.NewSupportBundleService(dataDir, logService, doctorService, sanitizer)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bundlePath, err := supportBundleService.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to create support bundle: %w", err)
	}

	fmt.Printf("Support bundle created: %s\n", bundlePath)

	// Handle --open flag
	if *openFlag {
		supportDir := bundlePath[:len(bundlePath)-len("/nrcc-support-"+time.Now().Format("20060102")+".zip")]
		return openDirectory(supportDir)
	}

	return nil
}

// openDirectory opens a directory in the default file manager
func openDirectory(dirPath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dirPath)
	case "linux":
		cmd = exec.Command("xdg-open", dirPath)
	case "windows":
		cmd = exec.Command("explorer", dirPath)
	default:
		return fmt.Errorf("unsupported OS for --open: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}

	return nil
}
