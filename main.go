package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/sync/semaphore"
)

func ReadLinesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func main() {
	// RequestList()
	// 日志保存到文件
	logFile, err := os.OpenFile("send.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("无法创建日志文件: %v\n", err)
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	lines, err := ReadLinesFromFile("request.txt")
	if err != nil {
		log.Printf("读取文件失败: %v", err)
	}

	// 配置参数
	maxConcurrency := 5            // 最大并发数
	maxInterval := 2 * time.Second // 最大间隔
	totalSends := len(lines)       // 设置总发送次数

	// 查找所有plist文件
	// files, err := filepath.Glob("*.plist")
	// if err != nil {
	// 	log.Fatalf("查找plist文件失败: %v", err)
	// }
	// if len(files) == 0 {
	// 	fmt.Println("未找到plist文件")
	// 	return
	// }

	client := resty.New()
	sem := semaphore.NewWeighted(int64(maxConcurrency))
	var wg sync.WaitGroup

	for i := 0; i < totalSends; i++ {
		wg.Add(1)
		if err := sem.Acquire(context.TODO(), 1); err != nil {
			log.Printf("获取信号量失败: %v", err)
			wg.Done()
			continue
		}

		go func() {
			defer wg.Done()
			defer sem.Release(1)

			// 随机选择一个文件
			index := rand.Intn(len(lines))
			data := lines[index]

			// data, err := os.ReadFile(f)
			// if err != nil {
			// 	log.Printf("读取文件失败: %s, 错误: %v", f, err)
			// 	return
			// }
			// base64 解码
			decoded, err := base64.StdEncoding.DecodeString(string(data))
			if err != nil {
				log.Printf("base64解码失败: %d, 错误: %v", index, err)
				return
			}

			resp, err := client.R().
				SetHeader("Content-Type", "application/xml").
				SetBody(decoded).
				Post("http://gs.apple.com/TSS/controller?action=2")
			if err != nil {
				log.Printf("Send Failed: %d, 错误: %v", index, err)
			} else if resp.IsSuccess() {
				log.Printf("Success: %d: %s\n", index, resp.String())
			} else {
				log.Printf("Send Failed: %d, 状态码: %d\n", index, resp.StatusCode())
			}

			// 随机间隔，最大不超过maxInterval
			time.Sleep(time.Duration(float64(maxInterval) * rand.Float64()))
		}()
	}

	wg.Wait()
	fmt.Println("全部发送完成")
}

func createRequestFile() ([]string, error) {
	RootDir := "C:\\Users\\jefferyz\\Downloads\\ITO-PC112_484D7EFB9DFD"
	filelist := make([]string, 0)
	err := filepath.WalkDir(RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".log" {
			fmt.Println("处理文件:", path)
			filelist = append(filelist, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("查找log文件失败: %v", err)
	}
	return filelist, err
}

func RequestList() {
	filelist, err := createRequestFile()
	if err != nil {
		fmt.Println("创建请求文件失败:", err)
		return
	}

	reqfile, err := os.Create("request.txt")
	if err != nil {
		return
	}
	defer reqfile.Close()

	for _, file := range filelist {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		scaner := bufio.NewScanner(f)
		var isData bool
		for scaner.Scan() {
			line := scaner.Text()
			if len(line) > 0 && strings.Contains(line, "tss request:<<<<<<<<<<") {
				isData = true
				continue
			}
			if isData {
				reqfile.WriteString(line + "\n")
				isData = false
			}
		}
		f.Close()
	}
}
