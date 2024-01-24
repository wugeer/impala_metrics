package utils

import (
	"impala_metrics/conf"
	"log"
	"os"
	"sync/atomic"
	"time"
)

func WatchConfigFile(configFilePath string, ticker *time.Ticker, config *atomic.Value, prometheusMetrics *atomic.Value, lastPrometheusMetrics *atomic.Value, logger *log.Logger) {
	var lastModTime time.Time

	for {
		select {
		case <-ticker.C:
			fileInfo, err := os.Stat(configFilePath)
			if err != nil {
				logger.Printf("Error getting file info: %v", err)
				continue
			}

			modTime := fileInfo.ModTime()
			if modTime.After(lastModTime) {
				logger.Println("Config file changed, reloading...")

				if err := conf.ReadConfig(configFilePath, logger, config, prometheusMetrics, lastPrometheusMetrics); err != nil {
					logger.Printf("Error reloading config: %v", err)
					continue
				}

				lastModTime = modTime
			}
		}
	}
}
