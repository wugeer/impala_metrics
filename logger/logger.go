package logger

import (
	"log"
	"os"

	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

// customLogger 是实现 io.Writer 接口的自定义日志器结构体
type customLogger struct {
	logger *log.Logger
}

func NewLogger() *log.Logger {
	cl := newCustomLogger()
	logger := log.New(cl, "", 0)
	return logger
}

func newCustomLogger() *customLogger {
	return &customLogger{
		logger: log.New(os.Stdout, "", 0),
	}
}

// Write 实现了 io.Writer 接口，用于自定义日志的输出格式
func (c *customLogger) Write(p []byte) (n int, err error) {
	// 获取调用者信息
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "???"
		line = 0
	}
	// 只获取文件名，不包括路径
	shortFile := filepath.Base(file)

	// 获取当前时间
	currentTime := time.Now().Format("2006-01-02 15:04:05.000")

	// 组合自定义的日志格式
	logMessage := fmt.Sprintf("%s %s:%d %s", currentTime, shortFile, line, string(p))
	// 写入日志并处理错误
	if err := c.logger.Output(2, logMessage); err != nil {
		return 0, err
	}
	return len(p), nil // 假定整个 p 都被处理了
}
