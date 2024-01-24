package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"impala_metrics/conf"
	logger2 "impala_metrics/logger"
	"impala_metrics/utils"
	"impala_metrics/worker"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	configFilePath        string
	safeConfig            atomic.Value
	prometheusMetrics     atomic.Value
	lastPrometheusMetrics atomic.Value
)

func parseFlags() {
	flag.StringVar(&configFilePath, "config-file", "config.yml", "the config file with yaml format.")
	flag.Parse()
}

func main() {
	// 初始化日志器
	logger := logger2.NewLogger()

	// 解析命令行参数
	parseFlags()

	if err := conf.ReadConfig(configFilePath, logger, &safeConfig, &prometheusMetrics, &lastPrometheusMetrics); err != nil {
		logger.Fatalf("解析YAML时发生错误: %v", err)
	}

	// 设置用于检查配置文件的定时器
	ticker := time.NewTicker(15 * time.Second) // 每15s检查一次配置文件
	//  检查配置文件是否发生变化
	go utils.WatchConfigFile(configFilePath, ticker, &safeConfig, &prometheusMetrics, &lastPrometheusMetrics, logger)

	config := safeConfig.Load().(*conf.Config)
	sleepInterval := time.Duration(config.IntervalSeconds) * time.Second
	httpServerPort := fmt.Sprintf(":%d", config.Port)
	logger.Printf("sleepInterval: %v \nlisten on port: %v\nmetrics:%v\nnumWorkers:%d\n",
		sleepInterval, httpServerPort, config.Metrics, config.NumWorkers)

	numWorkers := config.NumWorkers // 定义工作者数量
	logger.Printf("ready to start [%d] workers\n", numWorkers)
	jobs := make(chan worker.Job, len(config.Servers))
	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		go worker.Worker(w, jobs, &wg, logger, &prometheusMetrics)
	}

	logger.Println("Start to collect Servers metrics")
	go func() {
		for {
			config := safeConfig.Load().(*conf.Config)
			// 分发工作
			for _, server := range config.Servers {
				wg.Add(1)
				jobs <- worker.Job{Server: server}
			}
			wg.Wait()
			time.Sleep(time.Duration(config.IntervalSeconds) * time.Second)
		}
	}()

	// 调用处理信号的函数
	utils.HandleSignals(func() {
		// 关闭 jobs 通道，表示不再发送新工作
		close(jobs)
	}, logger)

	// 启动HTTP服务器
	logger.Printf("start to listen on port %v\n", httpServerPort)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(httpServerPort, nil); err != nil {
		logger.Fatalf("http.ListenAndServe(httpServerPort, nil) failed with %s\n", err)
	}
}
