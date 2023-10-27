package main

import (
	"bufio"
	"encoding/json"
	"github.com/fatih/color"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"path"
	"io"
	"fmt"
	"time"
	"net"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"context"
	"log"
	"path/filepath"

	"github.com/google/go-github/github"
)

type Config struct {
	Prompt          string   `json:"prompt"`
	InitialCommands []string `json:"initialCommands"`
	Theme           string   `json:"theme"`
	WgetEnabled     bool     `json:"wgetEnabled"`
}

type Theme struct {
	TextColor      string `json:"textColor"`
	BackgroundColor string `json:"backgroundColor"`
	PromptColor    string `json:"promptColor"`
	ErrorColor     string `json:"errorColor"`
	OutputColor    string `json:"outputColor"`
}
type PackageManager struct {
	client *github.Client
}

func NewPackageManager() *PackageManager {
	return &PackageManager{
		client: github.NewClient(nil),
	}
}
func DownloadFile(url string, filepath string) error {
	if _, err := os.Stat(filepath); err == nil {
		log.Printf("File %s already exists. Skipping download.\n", filepath)
		return nil
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return fmt.Errorf("Failed to check if file exists: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Failed to make a GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Non-OK HTTP status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("Failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to write to file: %w", err)
	}

	return nil
}
func (pm *PackageManager) Install(user string, repo string) error {
	log.Printf("Fetching releases for repository: %s/%s\n", user, repo)
	releases, _, err := pm.client.Repositories.ListReleases(context.Background(), user, repo, nil)
	if err != nil {
		return fmt.Errorf("Failed to list releases: %w", err)
	}

	if len(releases) == 0 {
		return fmt.Errorf("No releases found for repository %s/%s", user, repo)
	}

	latestRelease := releases[0]
	for _, asset := range latestRelease.Assets {
		if strings.HasSuffix(*asset.Name, ".exe") {
			log.Printf("Downloading asset: %s\n", *asset.Name)
			err = DownloadFile(asset.GetBrowserDownloadURL(), filepath.Join("packages", *asset.Name))
			if err != nil {
				return fmt.Errorf("Failed to download file: %w", err)
			}
			break
		}
	}

	return nil
}

func main() {
	config, theme := loadConfig()

	// Print welcome message and set terminal title
	getColor(theme.TextColor).Printf("\033]0;Cherry Terminal v1.0\007")
	getColor(theme.TextColor).Printf("Welcome to Cherry Terminal v1.0 beta\n\n")

	// Execute initial commands
	for _, command := range config.InitialCommands {
		executeCommand(command, theme)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		getColor(theme.PromptColor).Printf("%s ", config.Prompt)
		scanner.Scan()
		line := scanner.Text()
		executeCommand(line, theme)
	}
}

func getColor(colorName string) *color.Color {
	switch strings.ToLower(colorName) {
	case "red":
		return color.New(color.FgRed)
	case "green":
		return color.New(color.FgGreen)
	case "yellow":
		return color.New(color.FgYellow)
	case "blue":
		return color.New(color.FgBlue)
	case "magenta":
		return color.New(color.FgMagenta)
	case "cyan":
		return color.New(color.FgCyan)
	case "black":
		return color.New(color.FgBlack)
	case "white":
		return color.New(color.FgWhite)
	default:
		return color.New(color.FgWhite)
	}
}




// Implement wget, ls, help, and verfetch commands...

func wget(url string, filename ...string) {
	resp, err := http.Get(url)
	if err != nil {
		color.Red("%v", err)
		return
	}
	defer resp.Body.Close()

	// If no filename is provided, extract it from the URL
	file := ""
	if len(filename) > 0 {
		file = filename[0]
	} else {
		file = path.Base(url)
	}

	// Create the file
	out, err := os.Create(file)
	if err != nil {
		color.Red("%v", err)
		return
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		color.Red("%v", err)
		return
	}

	color.Green("File saved as %s", file)
}

func ls() {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		color.Red("%v", err)
		return
	}

	for _, file := range files {
		color.Yellow("%s", file.Name())
	}
}

func help() {
	color.Blue(`Cherry Terminal v1.0
	Available commands:
	- wget <url>: Fetches the content of <url> and prints it
	- ls: Lists the files in the current directory
	- help: Shows this help text
	- verfetch: Shows system information
	- ip: Prints the main IP address of the machine`)
}
func verfetch() {
	// Get host information
	hostStat, _ := host.Info()

	// Get CPU information
	cpuStat, _ := cpu.Info()

	// Get virtual memory
	vMem, _ := mem.VirtualMemory()

	// Get disk usage
	diskStat, _ := disk.Usage("/")

	// Print formatted output
	color.Magenta("Host: %s", hostStat.Hostname)
	color.Magenta("Operating System: %s", hostStat.OS)
	color.Magenta("Platform: %s", hostStat.Platform)
	color.Magenta("Platform Family: %s", hostStat.PlatformFamily)
	color.Magenta("Platform Version: %s", hostStat.PlatformVersion)
	color.Magenta("CPU: %s", cpuStat[0].ModelName)
	color.Magenta("Cores: %d", cpuStat[0].Cores)
	color.Magenta("Total Memory: %v GB", bToGb(vMem.Total))
	color.Magenta("Available Memory: %v GB", bToGb(vMem.Available))
	color.Magenta("Used Memory: %v GB", bToGb(vMem.Used))
	color.Magenta("Disk Total: %v GB", bToGb(diskStat.Total))
	color.Magenta("Disk Used: %v GB", bToGb(diskStat.Used))
	color.Magenta("Disk Free: %v GB", bToGb(diskStat.Free))
}

// Converts bytes to gigabytes
func bToGb(b uint64) uint64 {
    return b / (1024 * 1024 * 1024)
}

func hello() {
	fmt.Println("Hello, welcome to Cherry Terminal!")
}

func now() {
	currentTime := time.Now()
	fmt.Println("Current time: ", currentTime.Format("15:04:05"))
}
func printMainIP() {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		color.Red("Oops: %v\n", err.Error())
		return
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	color.Green("IP address: %s", localAddr.IP.String())
}


func createDefaultConfig() Config {
    config := Config{
        Prompt:          "$",
        InitialCommands: []string{},
        Theme:           "light",
        WgetEnabled:     true,
    }

    configJson, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        color.Red("Failed to create default config: %v\n", err)
        os.Exit(1)
    }

    err = ioutil.WriteFile("config.json", configJson, 0644)
    if err != nil {
        color.Red("Failed to write default config file: %v\n", err)
        os.Exit(1)
    }

    return config
}

func createDefaultThemes() {
    themes := map[string]Theme{
        "light": {
            TextColor:      "white",
            BackgroundColor: "white",
            PromptColor:    "blue",
            ErrorColor:     "red",
            OutputColor:    "green",
        },
        "dark": {
            TextColor:      "white",
            BackgroundColor: "black",
            PromptColor:    "cyan",
            ErrorColor:     "red",
            OutputColor:    "green",
        },
    }

    for themeName, theme := range themes {
        themeJson, err := json.MarshalIndent(theme, "", "  ")
        if err != nil {
            color.Red("Failed to create %s theme: %v\n", themeName, err)
            os.Exit(1)
        }

        err = ioutil.WriteFile("themes/" + themeName + ".json", themeJson, 0644)
        if err != nil {
            color.Red("Failed to write %s theme file: %v\n", themeName, err)
            os.Exit(1)
        }
    }
}


func loadConfig() (Config, Theme) {
    var config Config

    file, err := ioutil.ReadFile("config.json")
    if err != nil {
        // If the config file does not exist, create a default one
        if os.IsNotExist(err) {
            config = createDefaultConfig()
        } else {
            color.Red("Failed to read config file: %v\n", err)
            os.Exit(1)
        }
    } else {
        err = json.Unmarshal(file, &config)
        if err != nil {
            color.Red("Failed to parse config file: %v\n", err)
            os.Exit(1)
        }
    }

    theme := loadTheme(config.Theme)

    return config, theme
}

func loadTheme(themeName string) Theme {
    var theme Theme

    file, err := ioutil.ReadFile("themes/" + themeName + ".json")
    if err != nil {
        // If the theme file does not exist, create default ones
        if os.IsNotExist(err) {
            // Create directory if not exist
            if _, err := os.Stat("themes"); os.IsNotExist(err) {
                os.Mkdir("themes", 0755)
            }
            // Create default themes
            createDefaultThemes()
            return loadTheme(themeName)
        } else {
            color.Red("Failed to read theme file: %v", err)
            os.Exit(1)
        }
    } else {
        err = json.Unmarshal(file, &theme)
        if err != nil {
            color.Red("Failed to parse theme file: %v", err)
            os.Exit(1)
        }
    }

    return theme
}

func executeCommand(input string, theme Theme) {
	args := strings.Split(input, " ")

	switch strings.ToLower(args[0]) {
	case "exit":
		os.Exit(0)
	case "wget":
		if len(args) != 2 {
			getColor(theme.ErrorColor).Printf("wget command requires a URL\n")
			return
		}
		wget(args[1])
	case "ls":
		ls()
	case "help":
		help()
	case "verfetch":
		verfetch()
	case "ip": // add this case
		printMainIP()
	case "pkg":
		if len(args) < 3 {
			getColor(theme.ErrorColor).Printf("pkg command requires at least two arguments: install user/repo\n")
			return
		}
		if strings.ToLower(args[1]) != "install" {
			getColor(theme.ErrorColor).Printf("Unknown pkg command: %s\n", args[1])
			return
		}
		parts := strings.Split(args[2], "/")
		if len(parts) != 2 {
			getColor(theme.ErrorColor).Printf("Invalid repository format. It should be user/repo\n")
			return
		}
		pm := NewPackageManager()
		err := pm.Install(parts[0], parts[1])
		if err != nil {
			getColor(theme.ErrorColor).Printf("Failed to install: %v\n", err)
			return
		}
		getColor(theme.OutputColor).Printf("Package installation complete\n")
	default:
		cmd := exec.Command("cmd", "/C", input)
		output, err := cmd.CombinedOutput()
		if err != nil {
			getColor(theme.ErrorColor).Printf("Error: Invalid command! %v", "\n")
			return
		}
		getColor(theme.OutputColor).Printf("%s", string(output))
	}
}