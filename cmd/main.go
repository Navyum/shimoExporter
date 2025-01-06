package main

import (
	"Navyum/shimoExporter/internal/pkg/shimo"
	"flag"
	"fmt"

	"os"
)

func main() {

	/*
		go run Project/cmd/main.go \
		-rootpath="/path/to/download" \
		-exportType="pdf" \
		-shimoSid="your_shimo_sid_here" \
		-removeBlank=true
	*/

	// Define flags
	rootpath := flag.String("rootpath", "./download", "Path to the download directory")
	exportType := flag.String("exportType", "md", "Type of export (e.g., md, pdf, jpg, docx)")
	shimoSid := flag.String("shimoSid", "", "Shimo SID")
	removeBlank := flag.Bool("removeBlank", true, "option Remove title blanks")

	// Parse flags
	flag.Parse()

	SM := shimo.NewShimoWithOptions(
		shimo.WithExportType(*exportType),
		shimo.WithShimoSid(*shimoSid),
		shimo.WithRootPath(*rootpath),
		shimo.WithRemoveBlank(*removeBlank),
	)

	//创建路径为rootpath的文件夹
	if err := os.MkdirAll(*rootpath, 0755); err != nil {
		SM.Logger.Log(fmt.Sprintf("【错误】: 创建目录失败: %v", err))
		return
	}

	//检查shimo_sid是否设置
	if *shimoSid == "" {
		SM.Logger.Log("【错误】: shimo_sid未设置")
		return
	}

	// 检查shimoSid是否有效
	err := SM.CheckShimoSid()
	if err != nil {
		SM.Logger.Log("【错误】: shimo_sid无效")
		return
	}

	// 创建根目录
	root := &shimo.DirInfo{
		FileInfo: shimo.FileInfo{
			Path:  *rootpath,
			Id:    "",
			Title: "",
			Type:  "root",
		},
		Dirs:  nil,
		Files: nil,
	}
	SM.BuildStructureTree(root)
	SM.TraverseTree(root)
	fmt.Println("Download complete.")
}
