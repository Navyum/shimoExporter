package main

import (
	"errors"
	"fmt"
	"os"

	"Navyum/shimoExporter/internal/pkg/shimo"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var exportType string
var shimoSid string

// 定义日志接口
type Logger interface {
	Log(message string)
}

// 实现Logger接口的结构体
type AppLogger struct {
	logOutput          *widget.RichText
	logOutputContainer *container.Scroll
}

// Log方法实现
func (l *AppLogger) Log(message string) {
	l.logOutput.Segments = append(l.logOutput.Segments, &widget.TextSegment{
		Text: message,
	})
	//l.logOutput.SetText(l.logOutput.Text + message + "\n")
	l.logOutput.Refresh()
	l.logOutputContainer.ScrollToBottom()
}

func main() {
	var rootpath string
	var removeBlank bool
	isDownloading := false

	myApp := app.New()
	myWindow := myApp.NewWindow("石墨文档导出工具by Navyum")

	myApp.Settings().SetTheme(theme.DarkTheme()) // 设置为黑色主题

	// 创建用户下载地址输入框
	rootpathEntry := widget.NewEntry()
	rootpathEntry.SetPlaceHolder("请输入目标目录路径,需要自行创建")

	// 如果输入框为空，则设置为默认值
	if rootpathEntry.Text == "" {
		rootpathEntry.SetText("./download")
	}

	// 创建用户导出类型下拉选择框
	exportTypeSelect := widget.NewSelect([]string{"md", "pdf", "jpg", "docx"}, func(selected string) {
		// 处理选择逻辑
		exportType = selected
	})

	// 仅当没有选择时，设置默认值为md
	if exportTypeSelect.Selected == "" {
		exportTypeSelect.SetSelected("md") // 设置默认值为md
	}

	// 创建用户shimo_sid输入框
	shimoSidEntry := widget.NewEntry()
	shimoSidEntry.SetPlaceHolder("请输入shimo_sid")

	// 创建用户是否移除空格复选框
	removeBlankCheck := widget.NewCheck("移除文件名中的空格", func(checked bool) {
		// 处理复选框逻辑
		removeBlank = checked
	})

	// 默认勾选移除空格
	removeBlankCheck.Checked = true

	// 创建日志输出框
	logOutput := widget.NewRichText()

	// 创建日志输出容器
	logOutputContainer := container.NewScroll(logOutput)
	logOutputContainer.SetMinSize(fyne.NewSize(0, 400))

	// 创建开始下载按钮
	startButton := widget.NewButton("开始下载", func() {})

	// 创建日志输出类
	appLogger := &AppLogger{
		logOutput:          logOutput,
		logOutputContainer: logOutputContainer,
	}

	// 设置开始按钮为高亮
	startButton.Importance = widget.HighImportance

	// 设置开始按钮点击事件
	var SM *shimo.Shimo
	clickCount := 0

	startButton.OnTapped = func() {
		// 获取配置
		clickCount += 1
		rootpath = rootpathEntry.Text
		shimoSid = shimoSidEntry.Text
		exportType = exportTypeSelect.Selected
		removeBlank = removeBlankCheck.Checked

		// 创建shimo实例仅在首次点击时
		if SM == nil {
			// 创建shimo实例
			SM = shimo.NewShimoWithOptions(
				shimo.WithExportType(exportType),
				shimo.WithShimoSid(shimoSid),
				shimo.WithRootPath(rootpath),
				shimo.WithRemoveBlank(removeBlank),
				shimo.WithLogger(appLogger),
			)

			//创建路径为rootpath的文件夹
			if err := os.MkdirAll(rootpath, 0755); err != nil {
				SM.Logger.Log(fmt.Sprintf("【错误】: 创建目录失败: %v", err))
				dialog.ShowError(errors.New("创建目录失败"), myWindow)
				return
			}

			//检查shimo_sid是否设置
			if shimoSid == "" {
				SM.Logger.Log("【错误】: shimo_sid未设置")
				dialog.ShowError(errors.New("【shimo_sid】 未设置"), myWindow)
				return
			}

			// 检查shimoSid是否有效
			err := SM.CheckShimoSid()
			if err != nil {
				SM.Logger.Log("【错误】: shimo_sid无效")
				dialog.ShowError(errors.New("【shimo_sid】 格式错误或者无效"), myWindow)
				return
			}

			// 打印下载配置
			SM.Logger.Log(fmt.Sprintf("【下载配置如下】:\n  - 目标路径: %s\n  -导出类型: %s\n  -shimo_sid: %s\n  -移除空格: %v\n", rootpath, exportType, shimoSid, removeBlank))

			startButton.SetText("查询中，请稍后...")
			startButton.Disable()

			// 按钮设置为灰色
			startButton.Importance = widget.LowImportance

			// 核心逻辑处理

			// 创建根目录
			root := &shimo.DirInfo{
				FileInfo: shimo.FileInfo{
					Path:  rootpath,
					Id:    "",
					Title: "",
					Type:  "root",
				},
				Dirs:  nil,
				Files: nil,
			}
			SM.BuildStructureTree(root)
			SM.Logger.Log(fmt.Sprintf("【查询到文件数量】：%d\n", SM.FileCount))

			// 遍历文件结构树并下载文件
			startButton.Importance = widget.HighImportance
			startButton.SetText("暂停下载")
			startButton.Enable()
			go func() {
				SM.TraverseTree(root)
			}()
			isDownloading = true
		}

		// 再次点击，如果正在下载，则暂停下载
		if clickCount >= 2 {
			if isDownloading {

				startButton.SetText("继续下载")
				isDownloading = false
				// 发送暂停信号
				SM.PauseChan <- struct{}{}

				//再次点击，如果未正在下载，则发送信号继续下载
			} else {
				startButton.SetText("暂停下载")
				isDownloading = true
				// 发送继续下载信号
				SM.PauseChan <- struct{}{}
			}
		}
	}

	// 布局
	myWindow.SetContent(container.NewVBox(
		widget.NewLabel("导出路径:"),
		rootpathEntry,
		widget.NewLabel("导出类型:"),
		exportTypeSelect,
		widget.NewLabel("shimo_sid:"),
		shimoSidEntry,
		removeBlankCheck,
		startButton,
		appLogger.logOutputContainer,
	))

	myWindow.Resize(fyne.NewSize(400, 600)) // 设置窗口大小
	myWindow.ShowAndRun()

}
