package scanner

import (
	"context"
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"
)

// ANSI escape codes for text decoration and cursor control
const (
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorBlue      = "\033[34m"
	colorPurple    = "\033[35m"
	colorCyan      = "\033[36m"
	colorWhite     = "\033[37m"
	boldText       = "\033[1m"
	cursorSave     = "\033[s"    // Save cursor position
	cursorRestore  = "\033[u"    // Restore cursor position
	cursorBottom   = "\033[999B" // Move cursor to the bottom of the terminal
	clearLine      = "\033[2K"   // Clear the current line
)

func (s *Scanner) checkTarget(target Target, req *fasthttp.Request, resp *fasthttp.Response) string {
	// Enforce rate limiting
	if s.rateLimiter != nil {
		err := s.rateLimiter.Wait(context.Background())
		if err != nil {
			if s.config.Verbose {
				fmt.Printf("\n=== Rate Limit Error ===\n")
				fmt.Printf("Rate limit exceeded for %s: %v\n", target.IP, err)
				fmt.Printf("========================\n")
			}
			return ""
		}
	}

	req.Reset()
	resp.Reset()

	var results []string

	for _, protocol := range s.config.Protocols {
		reqURI := fmt.Sprintf("%s://%s%s", protocol, target.IP, target.Path)
		req.SetRequestURI(reqURI)
		req.SetHost(target.Hostname)
		req.Header.SetUserAgent("Mozilla/5.0 (X11; Linux x86_64)")
		req.Header.Set("X-Bug-Bounty", "h1-damian89-test")
		req.Header.Set("Connection", "close") // Force the server to close the connection

		hc := s.clients.getClient(target.IP, s.config)
		err := hc.DoTimeout(req, resp, s.config.RequestTimeout)
		if err != nil {
			if s.config.Verbose {
				fmt.Printf("\n=== Error ===\n")
				fmt.Printf("Failed to execute request to %s: %v\n", reqURI, err)
				fmt.Printf("========================\n")
			}
			continue
		}

		statusCode := resp.StatusCode()
		contentLength := resp.Header.Peek("Content-Length")
		body := resp.Body()
		title := extractTitle(body)

		// Handle redirects manually
		if s.config.FollowRedirects && (statusCode == 301 || statusCode == 302 || statusCode == 307 || statusCode == 308) {
			location := resp.Header.Peek("Location")
			if len(location) > 0 {
				redirectURI := string(location)
				if s.config.Verbose {
					fmt.Printf("[*] Following redirect to: %s\n", redirectURI)
				}
				req.SetRequestURI(redirectURI)
				err := hc.DoTimeout(req, resp, s.config.RequestTimeout)
				if err != nil {
					if s.config.Verbose {
						fmt.Printf("\n=== Redirect Error ===\n")
						fmt.Printf("Failed to follow redirect to %s: %v\n", redirectURI, err)
						fmt.Printf("========================\n")
					}
					continue
				}
				statusCode = resp.StatusCode()
				contentLength = resp.Header.Peek("Content-Length")
				body = resp.Body()
				title = extractTitle(body)
			}
		}

		if s.config.Verbose {
			fmt.Printf("\n=== Request ===\n")
			fmt.Printf("URI: %s\n", reqURI)
			fmt.Printf("Host: %s\n", target.Hostname)
			fmt.Printf("Method: %s\n", string(req.Header.Method()))
			req.Header.VisitAll(func(k, v []byte) {
				fmt.Printf("%s: %s\n", string(k), string(v))
			})

			fmt.Printf("\n=== Response ===\n")
			fmt.Printf("Status: %d\n", statusCode)
			fmt.Printf("Content-Length: %s\n", contentLength)
			fmt.Printf("Title: %s\n", title)
			resp.Header.VisitAll(func(k, v []byte) {
				fmt.Printf("%s: %s\n", string(k), string(v))
			})
			if len(body) > 0 {
				fmt.Printf("\nBody (truncated):\n%s\n", truncateString(string(body), 1000))
			}
			fmt.Printf("========================\n")
		}

		// Check if the status code is in the list of expected status codes
		if len(s.config.HTTPStatusIs) > 0 {
			found := false
			for _, code := range s.config.HTTPStatusIs {
				if statusCode == code {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if s.config.HTTPBodyIncludes != "" {
			if !strings.Contains(string(body), s.config.HTTPBodyIncludes) {
				continue
			}
		}

		// Decorate the output with colors and bold text
		results = append(results, fmt.Sprintf(
			"\n %s[+] Found match - IP: %s%s, Host: %s%s, Path: %s%s, Status: %s%d%s, Content-Length: %s%s%s, Title: %s%s%s",
			boldText+colorGreen,
			colorCyan, target.IP,
			colorYellow, target.Hostname,
			colorPurple, target.Path,
			colorBlue, statusCode, colorReset,
			colorRed, contentLength, colorReset,
			colorWhite, title, colorReset,
		))
	}

	if len(results) > 0 {
		return strings.Join(results, "\n")
	}
	return ""
}

func extractTitle(body []byte) string {
	bodyStr := string(body)
	titleStart := strings.Index(bodyStr, "<title>")
	if titleStart == -1 {
		return ""
	}
	titleStart += len("<title>")
	titleEnd := strings.Index(bodyStr[titleStart:], "</title>")
	if titleEnd == -1 {
		return bodyStr[titleStart:]
	}
	return bodyStr[titleStart : titleStart+titleEnd]
}

func truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen] + "..."
}

// Function to display the progress bar at the bottom of the terminal
func displayProgressBar(progress int, current, total int, elapsed, remaining string) {
	fmt.Print(cursorSave)    // Save the current cursor position
	fmt.Print(cursorBottom)  // Move the cursor to the bottom of the terminal
	fmt.Print(clearLine)     // Clear the current line
	fmt.Printf("[%d/%d] Scanning targets... %3d%% [%s%s] (%d/%d) [%s:%s]",
		current, total, progress,
		strings.Repeat("=", progress/2), strings.Repeat(" ", 50-progress/2),
		current, total, elapsed, remaining,
	)
	fmt.Print(cursorRestore) // Restore the cursor position
}

// Function to print results above the progress bar
func printResults(results string) {
	fmt.Print(cursorSave)    // Save the current cursor position
	fmt.Print(results)       // Print the results
	fmt.Print(cursorRestore) // Restore the cursor position
}
