package service

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
)

func TestChan(t *testing.T) {
	// nil chan
	//var ch chan int
	//go func() {
	//	ret, ok := <-ch
	//	fmt.Println(ret, ok)
	//}()
	//ch <- 1
	//time.Sleep(time.Second)
	//fmt.Println("done")
}

func Test_FileNotFound(t *testing.T) {
	path := "not_valid_file"
	_, err := os.Open(path)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			log.Println(path, "不存在")
			return
		}
		panic(err.Error())
	}
}

func Test_FileSeek(t *testing.T) {
	path := "test/SmallFile.log"
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Println("关闭文件句柄错误", path, err.Error())
		}
	}()
	buffer := bytes.NewBuffer(nil)
	cache := make([]byte, 64)
	var offset int64 = 10551
	ret, err := file.Seek(offset, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	if ret != offset {
		panic("seed位置错误")
	}
	index := 0
	for {
		read, err := file.Read(cache)
		if read <= 0 {
			if err != nil {
				if err == io.EOF {
					break
				}
				panic(err.Error())
			}
			break
		}
		start := 0
		cur := 0
		lines := make([]string, 0, 1)
		for i := 0; i < read; i++ {
			b := cache[i]
			// \n\r
			if b != '\r' && b != '\n' {
				continue
			}
			cur = i

			size := cur - start
			if buffer.Len() > 0 {
				size += buffer.Len()
			}
			if size == 0 {
				// 排除当前字节（因为是换行符）
				start = i + 1
				continue
			}
			if buffer.Len() > 0 {
				buffer.Write(cache[start:cur])
				line := buffer.String()
				lines = append(lines, line)
				buffer.Reset()
			} else {
				line := string(cache[start:cur])
				lines = append(lines, line)
			}

			start = i + 1
		}
		if start < read {
			buffer.Write(cache[start:read])
		}
		for _, v := range lines {
			index++
			fmt.Println(index, offset+int64(start), "["+v+"]")
		}
		offset += int64(read)
	}
	if buffer.Len() > 0 {
		panic("剩余内容" + buffer.String())
	}
}
