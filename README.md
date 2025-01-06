# shimoExporter
### 功能：从石墨文档全量导出文件到本地，支持导出格式md、docx、pdf、jpg

### UI模式
* 选择下载对应系统的可执行文件
* 按照UI提示输入信息，shimo_sid为必填参数
  - [x] 设置本地下载路径
  - [x] 设置导出格式
  - [x] 设置石墨sid
  - [x] 点击开始下载、暂停下载
  - [x] 支持实时查看日志信息

### 命令行模式
* 支持：定义本地路径，设置rootpath
* 支持：设置导出格式为,定义ExportType值，可选：pdf、jpg、docx、md，默认：md
* 支持设置是否去除标题中的空格
* 使用必须需要设置shimo_sid:
* 
  ```
    rootpath := "./download"    // 导出下载保存的地址，默认当前脚本执行路径下的download文件夹
  	exportType = "md"           // 导出文件保存类型: pdf、jpg、docx、md，默认md
	shimoSid = shimoSid         // 石墨cookie内的shimo_sid值
    removeBlank = true          // 是否删除标题中的空格，默认：false
  ```
* 命令行示例：
  ```bash
  	go run Project/cmd/main.go \
		-rootpath="/path/to/download" \
		-exportType="pdf" \
		-shimoSid="your_shimo_sid_here" \
		-removeBlank=true
  ```

### 获取石墨sid教程：
1. 浏览器F12——>找到导航栏 Application-> 选中左侧 Cookies-> 点击选中 https:shimo.im->点击右侧 shimo_sid->复制value
2. 图解：
<img width="1578" alt="image" src="https://github.com/Navyum/shimo_download/assets/36869790/9af9f5f0-65ec-4452-b863-b90da9c30281">

### 注意事项
* 本项目仅用于学习交流，如存在侵权行为，请联系删除

