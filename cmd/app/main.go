package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/1299172402/lanzou-upload-golang/lanzou"
)

func main() {
	file := flag.String("file", "", "要上传的文件路径")
	flag.Parse()

	if *file == "" {
		fmt.Println("用法: app -file <文件路径>")
		os.Exit(1)
	}

	cookie := os.Getenv("LANZOU_COOKIE")
	folderID := os.Getenv("LANZOU_FOLDER_ID")

	body, err := lanzou.Upload(*file, folderID, cookie)
	if err != nil {
		fmt.Println("上传失败:", err)
		os.Exit(1)
	}
	fmt.Println(body)
}
