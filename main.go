package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Config struct {
	Servers         []string `yaml:"servers"`
	IntervalSeconds int      `yaml:"intervalSeconds"`
	Port            int      `yaml:"port"`
	Metrics         []string `yaml:"metrics"`
}

var configFile string

type Metrics struct {
	Queue string
	Value float64
}

// NewMetrics 初始化Metrics
func NewMetrics(queue string, value float64) *Metrics {
	return &Metrics{
		Queue: queue,
		Value: value,
	}
}

func getMetricValue(url string, metricNames []string) (map[string][]*Metrics, error) {
	// 发送 HTTP GET 请求
	resp, err := http.Get(url)
	if err != nil {
		return map[string][]*Metrics{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatalf("resp.Body.Close() failed with %s\n", err)
		}
	}()

	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return map[string][]*Metrics{}, err
	}

	// 初始化指标值为未找到
	metricValue := map[string][]*Metrics{}

	// 在 HTML 中查找指标
	// 筛选table id为impala-server-tbl
	// 这里没有id选择器，因为不同的指标在不同的id下
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		// 检查每个 <tr> 元素中是否包含指标名称
		itemName := s.Find("td").First().Text()
		for _, metricName := range metricNames {
			if strings.HasPrefix(itemName, metricName) {
				// 获取紧接着指标名称的 <td> 元素中的文本，即为指标值
				queueName := strings.TrimPrefix(itemName, metricName+".")
				// 从url中获取ip
				//ip := strings.TrimPrefix(strings.Split(url, ":")[1], "//")
				originValue := strings.TrimSpace(s.Find("td").Eq(1).Text())
				actualValue, err := strconv.ParseFloat(originValue, 64)
				if err != nil {
					log.Fatalf("unable to case value %v to float64 with err %v", originValue, err)
				}
				metricValue[metricName] = append(metricValue[metricName], NewMetrics(queueName, actualValue))
				break
			}
		}
	})

	return metricValue, nil
}

var (
	cmdLine                             = flag.NewFlagSet("question", flag.ExitOnError)
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

func init() {
	cmdLine.StringVar(&configFile, "config-file", "config.yml", "the config file with yaml format.")
	prometheus.MustRegister(admissionControllerLocalMemAdmitted)
	prometheus.MustRegister(admissionControllerLocalNumQueued)
	prometheus.MustRegister(admissionControllerLocalNumAdmittedRunning)
}

func main() {

	if err := cmdLine.Parse(os.Args[1:]); err != nil {
		log.Fatalf("cmdLine.Parse(os.Args[1:]) failed with %s\n", err)
	}
	http.Handle("/metrics", promhttp.Handler())

	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("读取文件时发生错误: %v", err)
	}

	// 解析YAML文件
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("解析YAML时发生错误: %v", err)
	}

	sleepInterval := time.Duration(config.IntervalSeconds) * time.Second
	httpServerPort := fmt.Sprintf(":%d", config.Port)
	fmt.Printf("sleepInterval: %v \nlisten on port: %v\nmetrics:%v\n", sleepInterval, httpServerPort, config.Metrics)

	// 指定 URL 和指标名称
	//metricNames := []string{
	//	"admission-controller.local-mem-admitted",
	//	"admission-controller.local-num-queued",
	//	"admission-controller.local-num-admitted-running",
	//}
	metricNames := config.Metrics

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

	go func() {
		for {
			fmt.Println("Start to collect Servers metrics")
			for _, server := range config.Servers {
				// 将服务器地址和端口拼接成完整的 URL
				url := fmt.Sprintf("http://%v:25000/metrics", server)
				fmt.Printf("start to scrapy the metrics of server: %v\n", url)
				value, err := getMetricValue(url, metricNames)
				if err != nil {
					log.Fatal(err)
				}
				// fmt.Printf("The value of '%s' is: %s\n", metricNames, value)
				// 遍历输出的map类型指标
				for k, v := range value {
					for _, v1 := range v {
						prometheusMetricsNames[k].WithLabelValues(server, v1.Queue).Set(v1.Value)
					}
				}
			}
			time.Sleep(sleepInterval)
		}
	}()

	// 启动HTTP服务器
	fmt.Printf("start to listen on port %v\n", httpServerPort)
	log.Fatal(http.ListenAndServe(httpServerPort, nil))
}
