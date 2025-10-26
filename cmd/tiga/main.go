package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/app"
	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/version"
	"github.com/ysicing/tiga/static"

	_ "github.com/ysicing/tiga/docs/swagger" // Swagger docs
)

// @title Tiga DevOps Platform API
// @version 1.0
// @description Multi-instance DevOps management platform with support for MinIO, MySQL, PostgreSQL, Redis, Docker, and Caddy.
// @description
// @description Features:
// @description - Instance health monitoring and metrics collection
// @description - Alert management with rules and events
// @description - Comprehensive audit logging
// @description - MinIO object storage management
// @description - Database management with SQL editor
// @description - Docker container operations
// @description
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/ysicing/tiga
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name instances
// @tag.description Instance management operations

// @tag.name minio
// @tag.description MinIO object storage operations

// @tag.name alerts
// @tag.description Alert rules and events management

// @tag.name audit
// @tag.description Audit log operations

// @tag.name database
// @tag.description Database management operations

// @tag.name docker
// @tag.description Docker container operations

// @tag.name auth
// @tag.description Authentication operations

// @tag.name users
// @tag.description User management operations

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "Path to configuration file")

	// klog flags
	// klog.InitFlags removed
}

func main() {
	flag.Parse()

	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("Tiga Server\n")
		fmt.Printf("Version:    %s\n", version.Version)
		fmt.Printf("Build Time: %s\n", version.BuildTime)
		fmt.Printf("Commit ID:  %s\n", version.CommitID)
		os.Exit(0)
	}

	// Print version information in startup log
	logrus.WithFields(logrus.Fields{
		"version":    version.Version,
		"build_time": version.BuildTime,
		"commit_id":  version.CommitID,
	}).Info("Starting Tiga Server")

	// Check if system is installed
	configService := config.NewInstallConfigService(configFile)
	installMode := !configService.IsInstalled()

	if installMode {
		logrus.Info("System not initialized. Starting in installation mode...")
	}

	// Load or create minimal configuration
	var cfg *config.Config
	var err error
	if installMode {
		// Use default configuration for installation mode
		cfg = config.LoadFromEnv()
	} else {
		// Load configuration from file
		cfg, err = config.LoadFromFile(configFile)
		if err != nil {
			logrus.Fatalf("Failed to load configuration: %v", err)
		}
		logrus.Infof("Configuration loaded from %s", configFile)
	}

	// Create application
	application, err := app.NewApplication(cfg, configFile, installMode, static.FS)
	if err != nil {
		logrus.Fatalf("Failed to create application: %v", err)
	}

	// Initialize application
	ctx := context.Background()
	if err := application.Initialize(ctx); err != nil {
		logrus.Fatalf("Failed to initialize application: %v", err)
	}

	// Run application
	if err := application.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
