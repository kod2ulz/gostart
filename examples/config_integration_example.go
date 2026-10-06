package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kod2ulz/gostart/app"
	"github.com/kod2ulz/gostart/api/frameworks/gin"
	"github.com/kod2ulz/gostart/config"
)

func main() {
	// Define CLI flags
	configFormat := flag.String("dump-config", "", "Dump configuration and exit (formats: json, yaml, env)")
	showConfig := flag.Bool("show-config", false, "Alias for dump-config with YAML format")
	dumpCached := flag.Bool("dump-cached", false, "Dump only cached configuration (DB, Vault)")
	dumpEffective := flag.Bool("dump-effective", false, "Dump full effective configuration (all sources)")
	flag.Parse()

	// Load configuration
	if err := config.Yaml.Load("config.yaml"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load config.yaml: %v\n", err)
	}

	// Handle config dump requests
	if *showConfig {
		*configFormat = "yaml"
	}

	if *configFormat != "" {
		fmt.Printf("Configuration Dump (%s format):\n", *configFormat)
		fmt.Println("================================")

		if err := config.DumpConfig(config.ConfigFormat(*configFormat)); err != nil {
			fmt.Fprintf(os.Stderr, "Error dumping config: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *dumpCached {
		format := "yaml"
		if *configFormat != "" {
			format = *configFormat
		}

		fmt.Printf("Cached Configuration (%s format):\n", format)
		fmt.Printf("Vault cache: %d items\n", config.Vault.CacheSize())
		fmt.Printf("DB cache: %d items\n", config.DB.CacheSize())
		fmt.Println("================================")

		if err := config.DumpCachedConfig(config.ConfigFormat(format)); err != nil {
			fmt.Fprintf(os.Stderr, "Error dumping cached config: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *dumpEffective {
		format := "yaml"
		if *configFormat != "" {
			format = *configFormat
		}

		fmt.Printf("Effective Configuration (%s format):\n", format)
		fmt.Println("================================")
		fmt.Println("Shows all loaded config from YAML, ENV, DB (cached), and Vault (cached)")

		dump, err := config.CollectEffectiveConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error collecting config: %v\n", err)
			os.Exit(1)
		}

		var printErr error
		switch config.ConfigFormat(format) {
		case config.FormatJSON:
			printErr = dump.PrintJSON()
		case config.FormatYAML:
			printErr = dump.PrintYAML()
		case config.FormatENV:
			printErr = dump.PrintENV()
		}

		if printErr != nil {
			fmt.Fprintf(os.Stderr, "Error printing config: %v\n", printErr)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Normal application startup
	gin.Setup()

	application := app.Init(
		app.WithHeartbeatHandlers(),
	)

	application.Run()
}
