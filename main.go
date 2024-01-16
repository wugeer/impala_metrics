package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var configFilePath string

var (
	//cmdLine                             = flag.NewFlagSet("question", flag.ExitOnError)
	admissionControllerLocalMemAdmitted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admission_controller_local_mem_admitted",
			Help: "admission controller local mem admitted",
		},
		[]string{"ip", "queue"},
	)
	admissionControllerLocalNumQueued = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admission_controller_local_num_queued",
			Help: "admission controller local num queued",
		},
		[]string{"ip", "queue"},
	)
	admissionControllerLocalNumAdmittedRunning = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "admission_controller_local_num_admitted_running",
			Help: "admission controller local num admitted running",
		},
		[]string{"ip", "queue"},
	)
)

// job 表示一个工作项，这里我们定义为服务器的server地址
type job struct {
	server string
}

// worker 函数表示每个工作者如何处理工作
func worker(id int, jobs <-chan job, wg *sync.WaitGroup, logger *log.Logger, metricNames []string, prometheusMetricsNames map[string]*prometheus.GaugeVec) {
	logger.Printf("worker %d started\n", id)
	for j := range jobs {
		processMetrics(j.server, metricNames, prometheusMetricsNames, logger)
		wg.Done()
	}
}

// processMetrics 处理指标抓取和更新
func processMetrics(server string, metricNames []string, prometheusMetricsNames map[string]*prometheus.GaugeVec, logger *log.Logger) {
	url := fmt.Sprintf("http://%v:25000/metrics", server)
	logger.Printf("start to scrapy the metrics of server: %v\n", url)
	value, err := getMetricValue(url, metricNames, logger)
	if err != nil {
		log.Fatal(err)
	}
	// 遍历输出的map类型指标
	for k, v := range value {
		for _, v1 := range v {
			prometheusMetricsNames[k].WithLabelValues(server, v1.Queue).Set(v1.Value)
		}
	}
}

func init() {
	prometheus.MustRegister(admissionControllerLocalMemAdmitted)
	prometheus.MustRegister(admissionControllerLocalNumQueued)
	prometheus.MustRegister(admissionControllerLocalNumAdmittedRunning)
}

func parseFlags() {
	flag.StringVar(&configFilePath, "config-file", "config.yml", "the config file with yaml format.")
	flag.Parse()
}

func watchConfigFile(configFilePath string, ticker *time.Ticker, config *Config, logger *log.Logger) {
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
				newConfig, err := loadConfig(configFilePath, logger)
				if err != nil {
					logger.Printf("Error reloading config: %v", err)
					continue
				}

				*config = *newConfig
				logger.Printf("New config loaded: %+v", config)

				lastModTime = modTime
			}
		}
	}
}

func main() {
	// 初始化日志器
	cl := newCustomLogger()
	logger := log.New(cl, "", 0)

	// 解析命令行参数
	parseFlags()

	config, err := loadConfig(configFilePath, logger)
	if err != nil {
		logger.Fatalf("解析YAML时发生错误: %v", err)
	}

	// 设置用于检查配置文件的定时器
	ticker := time.NewTicker(15 * time.Second) // 每15s检查一次配置文件
	go watchConfigFile(configFilePath, ticker, config, logger)

	sleepInterval := time.Duration(config.IntervalSeconds) * time.Second
	httpServerPort := fmt.Sprintf(":%d", config.Port)
	logger.Printf("sleepInterval: %v \nlisten on port: %v\nmetrics:%v\nnumWorkers:%d\n",
		sleepInterval, httpServerPort, config.Metrics, config.NumWorkers)

	// 指定 URL 和指标名称
	metricNames := config.Metrics

	// 预定义所有指标名称
	allPrometheusMetricsNames := map[string]*prometheus.GaugeVec{
		"admission-controller.local-mem-admitted":         admissionControllerLocalMemAdmitted,
		"admission-controller.local-num-admitted-running": admissionControllerLocalNumAdmittedRunning,
		"admission-controller.local-num-queued":           admissionControllerLocalNumQueued,
	}

	prometheusMetricsNames := map[string]*prometheus.GaugeVec{}

	// 确定需要采集的指标
	for _, metricName := range metricNames {
		prometheusMetricsNames[metricName] = allPrometheusMetricsNames[metricName]
	}

	logger.Println("Start to collect Servers metrics")
	numWorkers := config.NumWorkers // 定义工作者数量
	jobs := make(chan job, len(config.Servers))
	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		go worker(w, jobs, &wg, logger, metricNames, prometheusMetricsNames)
	}

	go func() {
		for {
			// 分发工作
			for _, server := range config.Servers {
				wg.Add(1)
				jobs <- job{server: server}
			}
			wg.Wait()
			time.Sleep(time.Duration(config.IntervalSeconds) * time.Second)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// 启动处理信号的 goroutine
	go func() {
		sig := <-sigs
		logger.Println("接收到信号:", sig)
		// 关闭 jobs 通道，表示不再发送新工作
		close(jobs)
		// 可以在这里执行其他清理工作
		os.Exit(0)
	}()

	// 启动HTTP服务器
	logger.Printf("start to listen on port %v\n", httpServerPort)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(httpServerPort, nil); err != nil {
		logger.Fatalf("http.ListenAndServe(httpServerPort, nil) failed with %s\n", err)
	}
}
