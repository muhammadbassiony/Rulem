package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rulemig/internal/config"
)

// main is the entry point for the rulemig CLI tool.
func main() {
	cfgPath := filepath.Join(os.Getenv("HOME"), ".rulemig.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	fmt.Printf("Current storage directory: %s\n", cfg.StorageDir)
	fmt.Print("Would you like to set a new storage directory? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" || answer == "yes" {
		fmt.Print("Enter new storage directory path: ")
		newDir, _ := reader.ReadString('\n')
		newDir = strings.TrimSpace(newDir)
		if newDir != "" {
			if err := cfg.SetStorageDir(newDir, cfgPath); err != nil {
				fmt.Println("Failed to update storage directory:", err)
				os.Exit(1)
			}
			fmt.Println("Storage directory updated to:", newDir)
		}
	} else {
		fmt.Println("Using existing storage directory.")
	}
}
