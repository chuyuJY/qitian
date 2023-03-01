package log

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// ------------------------------- 第 1 步 -------------------------------
// [info ] 颜色为蓝色，[error] 为红色。
// 使用 log.Lshortfile 支持显示文件名和代码行号。
var (
	errorLog = log.New(os.Stdout, "\033[31m[error]\033[0m ", log.LstdFlags|log.Lshortfile)
	infoLog  = log.New(os.Stdout, "\033[34m[info ]\033[0m ", log.LstdFlags|log.Lshortfile)
	loggers  = []*log.Logger{errorLog, infoLog}
	mu       sync.Mutex
)

// 暴露 Error，Errorf，Info，Infof 4个方法。
var (
	Error  = errorLog.Println
	Errorf = errorLog.Printf
	Info   = infoLog.Println
	Infof  = infoLog.Printf
)

// ------------------------------- 第 2 步 -------------------------------
// 支持日志层级设置: InfoLevel, ErrorLevel, Disabled
const (
	InfoLevel = iota
	ErrorLevel
	Disabled
)

func SetLevel(level int) {
	mu.Lock()
	defer mu.Unlock()

	for _, logger := range loggers {
		logger.SetOutput(os.Stdout)
	}

	// 如果设置为 ErrorLevel，infoLog 的输出会被定向到 ioutil.Discard，即不打印该日志
	if ErrorLevel < level {
		errorLog.SetOutput(ioutil.Discard)
	}
	if InfoLevel < level {
		infoLog.SetOutput(ioutil.Discard)
	}
}
