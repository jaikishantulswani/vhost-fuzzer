package scanner

import (
	"bufio"
	"os"
)

const (
	batchSize     = 1000
	bufferSize    = 1024 * 1024 // 1MB buffer
	progressBatch = 10000       // Update progress every 10K requests
)

type BatchProcessor struct {
	ipFile     *os.File
	hostFile   *os.File
	paths      []string
	targetChan chan Target
	batchSize  int
}

func NewBatchProcessor(ipPath, hostPath string, paths []string, targetChan chan Target) (*BatchProcessor, error) {
	ipFile, err := os.Open(ipPath)
	if err != nil {
		return nil, err
	}

	hostFile, err := os.Open(hostPath)
	if err != nil {
		ipFile.Close()
		return nil, err
	}

	return &BatchProcessor{
		ipFile:     ipFile,
		hostFile:   hostFile,
		paths:      paths,
		targetChan: targetChan,
		batchSize:  batchSize,
	}, nil
}

func (bp *BatchProcessor) Close() {
	bp.ipFile.Close()
	bp.hostFile.Close()
}

func (bp *BatchProcessor) ProcessFiles() error {
	defer close(bp.targetChan)

	ipScanner := bufio.NewScanner(bp.ipFile)
	ipScanner.Buffer(make([]byte, bufferSize), bufferSize)

	for ipScanner.Scan() {
		ip := ipScanner.Text()
		if ip == "" {
			continue
		}

		if err := bp.processIPWithHosts(ip); err != nil {
			return err
		}
	}

	return ipScanner.Err()
}

func (bp *BatchProcessor) processIPWithHosts(ip string) error {
	_, err := bp.hostFile.Seek(0, 0)
	if err != nil {
		return err
	}

	hostScanner := bufio.NewScanner(bp.hostFile)
	hostScanner.Buffer(make([]byte, bufferSize), bufferSize)

	for hostScanner.Scan() {
		host := hostScanner.Text()
		if host == "" {
			continue
		}

		for _, path := range bp.paths {
			bp.targetChan <- Target{
				IP:       ip,
				Hostname: host,
				Path:     path,
			}
		}
	}

	return hostScanner.Err()
}
