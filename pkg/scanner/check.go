package scanner

import (
	"context" // Add this import
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"
)

func (s *Scanner) checkTarget(target Target, req *fasthttp.Request, resp *fasthttp.Response) string {
	// Enforce rate limiting
	if s.rateLimiter != nil {
		err := s.rateLimiter.Wait(context.Background()) // Wait for a token from the rate limiter
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

	reqURI := fmt.Sprintf("%s://%s%s", s.config.Protocol, target.IP, target.Path)
	req.SetRequestURI(reqURI)
	req.SetHost(target.Hostname)
	req.Header.SetUserAgent("Mozilla/5.0 (X11; Linux x86_64)")
	req.Header.Set("X-Bug-Bounty", "h1-damian89-test")
	req.Header.SetBytesKV([]byte("Connection"), []byte("close"))

	hc := s.clients.getClient(target.IP, s.config)
	err := hc.DoTimeout(req, resp, s.config.RequestTimeout)
	if err != nil {
		if s.config.Verbose {
			fmt.Printf("\n=== Error ===\n")
			fmt.Printf("Failed to execute request to %s: %v\n", reqURI, err)
			fmt.Printf("========================\n")
		}
		return ""
	}

	statusCode := resp.StatusCode()
	contentLength := resp.Header.Peek("Content-Length")
	body := resp.Body()
	title := extractTitle(body)

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
			return ""
		}
	}

	if s.config.HTTPBodyIncludes != "" {
		if !strings.Contains(string(body), s.config.HTTPBodyIncludes) {
			return ""
		}
	}

	return fmt.Sprintf("\n[+] Found match - IP: %s, Host: %s, Path: %s, Status: %d, Content-Length: %s, Title: %s",
		target.IP, target.Hostname, target.Path, statusCode, contentLength, title)
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
		return ""
	}
	return bodyStr[titleStart : titleStart+titleEnd]
}

func truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen] + "..."
}
