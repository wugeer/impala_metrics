package utils

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// HandleSignals 创建一个处理操作系统信号的 goroutine。
func HandleSignals(cleanupFunc func(), logger *log.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Printf("接收到信号: %v", sig)

		// 执行任何清理操作
		if cleanupFunc != nil {
			cleanupFunc()
		}

		os.Exit(0)
	}()
}
