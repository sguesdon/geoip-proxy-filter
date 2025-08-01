package main

import (
	"fmt"
	"os"

	"github.com/sguesdon/geoip-proxy-filter/internal/config"
	"github.com/sguesdon/geoip-proxy-filter/internal/geoip"
)

func main() {
	config := config.NewConfig()

	downloader := geoip.NewDownloader(config)
	if err := downloader.UpdateDatabase(); err != nil {
		fmt.Printf("Error updating database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Geoip database updated successfully")
}