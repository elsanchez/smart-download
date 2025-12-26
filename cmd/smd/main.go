package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/elsanchez/smart-download/pkg/client"
)

const (
	version = "0.1.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Crear cliente
	c := client.NewDefaultClient()

	switch os.Args[1] {
	case "add":
		handleAdd(c, os.Args[2:])
	case "status":
		handleStatus(c, os.Args[2:])
	case "list":
		handleList(c, os.Args[2:])
	case "stats":
		handleStats(c)
	case "version":
		fmt.Printf("smd v%s\n", version)
	case "help":
		printUsage()
	default:
		// Si el primer argumento parece una URL, asumir que es "add"
		if len(os.Args[1]) > 4 && (os.Args[1][:4] == "http") {
			handleAdd(c, os.Args[1:])
		} else {
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println(`Smart Media Downloader (smd) v` + version + `

Usage: smd <command> [args]

Commands:
  add <url> [options]  Add download to queue
  status <id>          Get download status
  list [limit]         List recent downloads (default: 50)
  stats                Show queue statistics
  version              Show version
  help                 Show this help

Add Options:
  --clip <start> <end>  Clip video segment (format: HH:MM:SS or seconds)
  --gif [width]         Convert to GIF (default width: 480px)
  --no-convert          Skip auto-conversion to WhatsApp MP4
  --resolution <res>    Video resolution (1080p, 720p, 480p)
  --audio-only          Extract audio only

Examples:
  smd add https://youtube.com/watch?v=xxx
  smd add https://youtube.com/watch?v=xxx --clip 00:10 00:30
  smd add https://youtube.com/watch?v=xxx --gif 480
  smd add https://youtube.com/watch?v=xxx --no-convert
  smd https://youtube.com/watch?v=xxx          (shorthand for 'add')
  smd status 123
  smd list 10
  smd stats`)
}

func handleAdd(c *client.Client, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: URL is required")
		printUsage()
		os.Exit(1)
	}

	// Parse flags
	addFlags := flag.NewFlagSet("add", flag.ExitOnError)
	clipStart := addFlags.String("clip-start", "", "Clip start time (HH:MM:SS or seconds)")
	clipEnd := addFlags.String("clip-end", "", "Clip end time (HH:MM:SS or seconds)")
	convertToGIF := addFlags.Bool("gif", false, "Convert to GIF")
	gifWidth := addFlags.Int("gif-width", 480, "GIF width in pixels")
	noConvert := addFlags.Bool("no-convert", false, "Skip WhatsApp MP4 conversion")
	resolution := addFlags.String("resolution", "", "Video resolution (1080p, 720p, 480p)")
	audioOnly := addFlags.Bool("audio-only", false, "Extract audio only")

	// URL es el primer argumento
	url := args[0]

	// Parse remaining args
	if len(args) > 1 {
		addFlags.Parse(args[1:])
	}

	// Validación de clip
	if (*clipStart != "" && *clipEnd == "") || (*clipStart == "" && *clipEnd != "") {
		fmt.Println("Error: Both --clip-start and --clip-end are required for clipping")
		os.Exit(1)
	}

	// Construir options
	options := make(map[string]interface{})

	if *resolution != "" {
		options["resolution"] = *resolution
	}
	if *audioOnly {
		options["audio_only"] = true
	}
	if *clipStart != "" && *clipEnd != "" {
		options["clip_start"] = *clipStart
		options["clip_end"] = *clipEnd
	}
	if *convertToGIF {
		options["convert_to_gif"] = true
		if *gifWidth > 0 {
			options["gif_width"] = *gifWidth
		}
	}
	if *noConvert {
		options["no_convert"] = true
	}

	payload := &client.AddDownloadPayload{
		URL:     url,
		Options: options,
	}

	id, err := c.AddDownload(payload)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Download added with ID: %d\n", id)
	fmt.Printf("  URL: %s\n", url)

	// Mostrar opciones configuradas
	if len(options) > 0 {
		fmt.Println("  Options:")
		if *clipStart != "" {
			fmt.Printf("    Clip: %s - %s\n", *clipStart, *clipEnd)
		}
		if *convertToGIF {
			fmt.Printf("    GIF: %dpx width\n", *gifWidth)
		}
		if *noConvert {
			fmt.Println("    Skip WhatsApp conversion")
		}
		if *resolution != "" {
			fmt.Printf("    Resolution: %s\n", *resolution)
		}
		if *audioOnly {
			fmt.Println("    Audio only")
		}
	}

	fmt.Println("  Status: pending")
}

func handleStatus(c *client.Client, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Download ID is required")
		fmt.Println("Usage: smd status <id>")
		os.Exit(1)
	}

	var id int64
	if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
		fmt.Printf("Error: Invalid ID: %s\n", args[0])
		os.Exit(1)
	}

	status, err := c.GetDownloadStatus(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", status)
}

func handleList(c *client.Client, args []string) {
	limit := 50
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &limit); err != nil {
			fmt.Printf("Error: Invalid limit: %s\n", args[0])
			os.Exit(1)
		}
	}

	downloads, err := c.ListRecentDownloads(limit)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(downloads) == 0 {
		fmt.Println("No downloads found")
		return
	}

	fmt.Printf("Recent downloads (%d):\n\n", len(downloads))

	for _, dl := range downloads {
		id := int64(dl["id"].(float64))
		url := dl["url"].(string)
		status := dl["status"].(string)
		platform := dl["platform"].(string)

		fmt.Printf("ID: %d\n", id)
		fmt.Printf("  Platform: %s\n", platform)
		fmt.Printf("  URL: %s\n", url)
		fmt.Printf("  Status: %s\n", status)

		if outputPath, ok := dl["output_path"].(string); ok && outputPath != "" {
			fmt.Printf("  Output: %s\n", outputPath)
		}

		if errMsg, ok := dl["error_message"].(string); ok && errMsg != "" {
			fmt.Printf("  Error: %s\n", errMsg)
		}

		fmt.Println()
	}
}

func handleStats(c *client.Client) {
	payload, err := c.Send(&client.Request{
		Action: "stats",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !payload.Success {
		fmt.Printf("Error: %s\n", payload.Error)
		os.Exit(1)
	}

	fmt.Println("Queue Statistics:")
	fmt.Println()

	// Parse stats
	var stats map[string]interface{}
	if err := json.Unmarshal(payload.Data, &stats); err != nil {
		fmt.Printf("Error parsing stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Pending:      %d\n", int(stats["pending"].(float64)))
	fmt.Printf("  Downloading:  %d\n", int(stats["downloading"].(float64)))
	fmt.Printf("  Processing:   %d\n", int(stats["processing"].(float64)))
	fmt.Printf("  Completed:    %d\n", int(stats["completed"].(float64)))
	fmt.Printf("  Failed:       %d\n", int(stats["failed"].(float64)))
	fmt.Println()
	fmt.Printf("  Workers:      %d / %d busy\n", int(stats["workers_busy"].(float64)), int(stats["workers_total"].(float64)))
}
