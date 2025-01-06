package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"strconv"
)

type Config struct {
	IPsFile             string
	HostsFile           string
	Concurrency         int
	Paths               []string
	HTTPBodyIncludes    string
	HTTPStatusIs        []int // Change to a slice of integers
	Verbose             bool
	RequestTimeout      time.Duration
	MaxIdleConnDuration time.Duration
	MaxConnDuration     time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	Protocol            string
	RateLimit           int
}

func ParseFlags() Config {
	config := Config{}
	var pathsStr string
	var protocol string
	var requestTimeout, maxIdleConnDuration, maxConnDuration, readTimeout, writeTimeout int
	var httpStatusIsStr string // Add this variable to hold the comma-separated status codes

	flag.StringVar(&config.IPsFile, "ips", "", "File containing IP addresses")
	flag.StringVar(&config.HostsFile, "hosts", "", "File containing hostnames")
	flag.IntVar(&config.Concurrency, "concurrency", 100, "Number of concurrent requests")
	flag.StringVar(&pathsStr, "paths", "/", "Comma-separated list of paths to check")
	flag.StringVar(&protocol, "protocol", "http", "http/https")
	flag.StringVar(&config.HTTPBodyIncludes, "http-body-includes", "", "String to search for in response body")
	flag.StringVar(&httpStatusIsStr, "http-status-is", "", "Comma-separated list of expected HTTP status codes") // Update this flag
	flag.IntVar(&requestTimeout, "request-timeout", 4, "Timeout for individual requests in seconds")
	flag.IntVar(&maxIdleConnDuration, "max-idle-timeout", 6, "Maximum idle connection duration in seconds")
	flag.IntVar(&maxConnDuration, "max-conn-timeout", 6, "Maximum connection duration in seconds")
	flag.IntVar(&readTimeout, "read-timeout", 5, "Read timeout in seconds")
	flag.IntVar(&writeTimeout, "write-timeout", 5, "Write timeout in seconds")
	flag.BoolVar(&config.Verbose, "verbose", false, "Show all requests and responses")
	flag.IntVar(&config.RateLimit, "rate-limit", 0, "Rate limit in requests per second (0 for no limit)")

	flag.Parse()

	if config.IPsFile == "" || config.HostsFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Parse the comma-separated status codes into a slice of integers
	if httpStatusIsStr != "" {
		statusCodes := strings.Split(httpStatusIsStr, ",")
		for _, codeStr := range statusCodes {
			code, err := strconv.Atoi(strings.TrimSpace(codeStr))
			if err != nil {
				fmt.Printf("Invalid HTTP status code: %s\n", codeStr)
				os.Exit(1)
			}
			config.HTTPStatusIs = append(config.HTTPStatusIs, code)
		}
	}

	config.RequestTimeout = time.Duration(requestTimeout) * time.Second
	config.MaxIdleConnDuration = time.Duration(maxIdleConnDuration) * time.Second
	config.MaxConnDuration = time.Duration(maxConnDuration) * time.Second
	config.ReadTimeout = time.Duration(readTimeout) * time.Second
	config.WriteTimeout = time.Duration(writeTimeout) * time.Second

	config.Paths = strings.Split(pathsStr, ",")
	for i, path := range config.Paths {
		if !strings.HasPrefix(path, "/") {
			config.Paths[i] = "/" + path
		}
	}

	protocol = strings.ToLower(protocol)
	if protocol != "http" && protocol != "https" {
		fmt.Printf("Unknown protocol '%s'. Falling back to http.\n", protocol)
		protocol = "http"
	}
	config.Protocol = protocol

	return config
}
