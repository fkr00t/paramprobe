package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
	"github.com/fatih/color"
)

const (
	REFLECTION_MARKER = "PAYLOAD"
	REPO_URL          = "github.com/fkr00t/paramprobe" // Ganti dengan URL repository Anda
)

var (
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 10; SM-A505FN) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
	}
)

func printBanner() {
	banner := `
▛▀▖             ▛▀▖      ▌     
▙▄▘▝▀▖▙▀▖▝▀▖▛▚▀▖▙▄▘▙▀▖▞▀▖▛▀▖▞▀▖
▌  ▞▀▌▌  ▞▀▌▌▐ ▌▌  ▌  ▌ ▌▌ ▌▛▀ 
▘  ▝▀▘▘  ▝▀▘▘▝ ▘▘  ▘  ▝▀ ▀▀ ▝▀▘

    Version: 1.0.0
    Reflected Parameter Finder
    Author: fkr00t | Github: https://github.com/fkr00t
    `
	color.Red(banner)
}

func fetchURL(target string, userAgent string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // Set timeout untuk HTTP request
	}

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func isInternalURL(u, targetDomain string) bool {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return false
	}
	return strings.HasSuffix(parsedURL.Host, targetDomain)
}

func crawlDomain(target string, crawlSubdomains bool, delay time.Duration, userAgent string) (map[string][]string, string, error) {
	color.Cyan("[*] Starting domain crawl...")
	crawledURLs := make(map[string]bool)
	parameters := make(map[string][]string)
	toVisit := []string{target}

	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, "", err
	}
	targetDomain := targetURL.Host

	// Create a "result" folder if it doesn't exist
	resultFolder := "result"
	if err := os.MkdirAll(resultFolder, os.ModePerm); err != nil {
		return nil, "", err
	}

	// Create a subfolder inside "result" for the target domain
	targetFolder := filepath.Join(resultFolder, strings.ReplaceAll(targetDomain, ".", "_"))
	if err := os.MkdirAll(targetFolder, os.ModePerm); err != nil {
		return nil, "", err
	}

	for len(toVisit) > 0 {
		url := toVisit[0]
		toVisit = toVisit[1:]

		if crawledURLs[url] {
			continue
		}
		crawledURLs[url] = true

		response, err := fetchURL(url, userAgent)
		if err != nil {
			continue
		}

		pageFilename := filepath.Join(targetFolder, strings.ReplaceAll(targetURL.Path, "/", "_")+"index.html")
		if err := ioutil.WriteFile(pageFilename, []byte(response), 0644); err != nil {
			return nil, "", err
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(response))
		if err != nil {
			continue
		}

		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			fullURL, err := targetURL.Parse(href)
			if err != nil {
				return
			}

			if crawlSubdomains || isInternalURL(fullURL.String(), targetDomain) {
				toVisit = append(toVisit, fullURL.String())
			}

			queryParams := fullURL.Query()
			for param := range queryParams {
				baseURL := fullURL.Scheme + "://" + fullURL.Host + fullURL.Path
				parameters[baseURL] = append(parameters[baseURL], param)
			}
		})

		// Tambahkan delay antara setiap permintaan
		time.Sleep(delay)
	}

	color.Cyan("[+] Successfully crawled %d unique pages.", len(crawledURLs))
	color.Cyan("[+] Found %d unique parameters.", len(parameters))

	return parameters, targetFolder, nil
}

func checkReflectedParameter(baseURL, param string, userAgent string) string {
	testValue := REFLECTION_MARKER
	query := url.Values{}
	query.Set(param, testValue)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", baseURL+"?"+query.Encode(), nil)
	if err != nil {
		return ""
	}

	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	if strings.Contains(string(body), testValue) {
		return baseURL + "?" + query.Encode()
	}
	return ""
}

const (
	REPO_URL = "github.com/fkr00t/paramprobe" // Clean package path
)

func updateTool() {
	color.Cyan("[*] Checking for updates...")

	// Jalankan perintah `go install` untuk mengupdate tools
	cmd := exec.Command("go", "install", REPO_URL+"@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		color.Red("[!] Failed to update tool: %v", err)
		return
	}

	color.Green("[+] Tool updated successfully!")
}

func main() {
	printBanner()

	// Define flags
	target := flag.String("u", "", "Target URL to crawl (e.g., http://example.com)")
	flag.StringVar(target, "url", "", "Target URL to crawl (e.g., http://example.com)")
	crawlSubdomains := flag.Bool("c", false, "Crawl subdomains as well")
	flag.BoolVar(crawlSubdomains, "crawl", false, "Crawl subdomains as well")
	delay := flag.Duration("d", 0, "Delay between requests (e.g., 1s, 500ms)")
	flag.DurationVar(delay, "delay", 0, "Delay between requests (e.g., 1s, 500ms)")
	userAgent := flag.String("user-agent", "", "Custom User-Agent string")
	randomAgent := flag.Bool("random-agent", false, "Use a random User-Agent")
	update := flag.Bool("up", false, "Update the tool to the latest version")
	flag.BoolVar(update, "update", false, "Update the tool to the latest version")
	help := flag.Bool("h", false, "Show help message")
	flag.BoolVar(help, "help", false, "Show help message")

	// Custom help message
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of ParamProbe:\n")
		fmt.Println("  -u, --url string")
		fmt.Println("        Target URL to crawl (e.g., http://example.com)")
		fmt.Println("  -c, --crawl")
		fmt.Println("        Crawl subdomains as well (optional)")
		fmt.Println("  -d, --delay duration")
		fmt.Println("        Delay between requests (e.g., 1s, 500ms)")
		fmt.Println("  --user-agent string")
		fmt.Println("        Custom User-Agent string")
		fmt.Println("  --random-agent")
		fmt.Println("        Use a random User-Agent")
		fmt.Println("  -up, --update")
		fmt.Println("        Update the tool to the latest version")
		fmt.Println("  -h, --help")
		fmt.Println("        Show this help message")
		fmt.Println("\nExample:")
		fmt.Println("  paramprobe -u http://testphp.vulnweb.com -d 1s --random-agent")
		fmt.Println("  paramprobe -u http://testphp.vulnweb.com --user-agent 'MyCustomAgent'")
		fmt.Println("  paramprobe --update")
	}

	// Parse flags
	flag.Parse()

	// Show help message if -h or --help is used
	if *help {
		flag.Usage()
		return
	}

	// Jika opsi --update digunakan, update tools dan keluar
	if *update {
		updateTool()
		return
	}

	// Validate target flag
	if *target == "" {
		color.Red("[!] Target URL is required. Use -h for help.")
		return
	}

	if !strings.HasPrefix(*target, "http://") && !strings.HasPrefix(*target, "https://") {
		color.Red("[!] Target URL must start with http:// or https://")
		return
	}

	// Set User-Agent
	selectedUserAgent := ""
	if *randomAgent {
		selectedUserAgent = userAgents[rand.Intn(len(userAgents))]
		color.Cyan("[*] Using random User-Agent: %s", selectedUserAgent)
	} else if *userAgent != "" {
		selectedUserAgent = *userAgent
		color.Cyan("[*] Using custom User-Agent: %s", selectedUserAgent)
	} else {
		selectedUserAgent = userAgents[0] // Default User-Agent
	}

	// Start progress bar
	color.Cyan("[*] Starting process...")
	if *crawlSubdomains {
		color.Cyan("[*] Crawling subdomains is enabled.")
	}
	if *delay > 0 {
		color.Cyan("[*] Delay between requests: %v", *delay)
	}

	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to listen for interrupt signals
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	// Handle interrupt signal
	go func() {
		<-interruptChan
		color.Yellow("\n[!] Process interrupted by user. Cleaning up...")
		cancel() // Batalkan context
		os.Exit(1)
	}()

	// Step 1: Crawl the domain
	parameters, targetFolder, err := crawlDomain(*target, *crawlSubdomains, *delay, selectedUserAgent)
	if err != nil {
		color.Red("[!] Error crawling domain: %v", err)
		return
	}

	// Step 2: Test reflected parameters
	color.Cyan("[*] Testing for reflected parameters...")
	if *crawlSubdomains {
		color.Cyan("[*] Including subdomains in the test.")
	}

	// Initialize progress bar
	bar := pb.New(len(parameters))
	bar.SetTemplateString(`{{with string . "prefix"}}{{.}} {{end}}{{counters . }} {{bar . }} {{percent . }} {{etime . "ETA: %s"}}`)
	bar.Set("prefix", "Progress:")
	bar.Start()

	reflectedResults := make(map[string]bool) // Use a map to store unique results

	var wg sync.WaitGroup
	var mu sync.Mutex

	for baseURL, params := range parameters {
		for _, param := range params {
			wg.Add(1)
			go func(baseURL, param string) {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return // Berhenti jika context dibatalkan
				default:
					result := checkReflectedParameter(baseURL, param, selectedUserAgent)
					if result != "" {
						mu.Lock()
						reflectedResults[result] = true // Store unique results in the map
						mu.Unlock()
					}
					bar.Increment()
				}
			}(baseURL, param)
		}
	}

	wg.Wait()
	bar.Finish()

	// Output results
	if len(reflectedResults) > 0 {
		color.Green("\n[+] Reflected Parameters Found:")
		for result := range reflectedResults {
			color.Green("  [Reflected] %s", result)
			resultFile := filepath.Join(targetFolder, "reflected_parameters.txt")
			f, err := os.OpenFile(resultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				color.Red("[!] Error writing to file: %v", err)
				continue
			}
			if _, err := f.WriteString(result + "\n"); err != nil {
				color.Red("[!] Error writing to file: %v", err)
			}
			f.Close()
		}
	} else {
		color.Red("\n[-] No reflected parameters found.")
	}
}
