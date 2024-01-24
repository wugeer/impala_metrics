package metrics

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"strings"

	"log"
)

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

// CreateGaugeVec 根据提供的指标名称和帮助文本创建一个新的 prometheus.GaugeVec。
func CreateGaugeVec(metricName string, helpText string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricName,
			Help: helpText,
		},
		[]string{"ip", "queue"},
	)
}

// GetMetricValue 从指定url中获取对应的指标值
func GetMetricValue(url string, metricNames []string, myLogger *log.Logger) (map[string][]*Metrics, error) {
	// 发送 HTTP GET 请求
	resp, err := http.Get(url)
	if err != nil {
		myLogger.Printf("Error retrieving metrics from %s: %v", url, err)
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			myLogger.Printf("resp.Body.Close() failed with %s\n", err)
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
					myLogger.Printf("unable to case value %v to float64 with err %v", originValue, err)
				}
				metricValue[metricName] = append(metricValue[metricName], NewMetrics(queueName, actualValue))
				break
			}
		}
	})

	return metricValue, nil
}
