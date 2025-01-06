package shimo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Define the ShimoUrl struct
type ShimoUrl struct {
	Root   string
	List   string
	Export string
	Query  string
}

var smUrl = ShimoUrl{
	Root:   "https://shimo.im/lizard-api/files",
	List:   "https://shimo.im/lizard-api/files?folder=%s",
	Export: "https://shimo.im/lizard-api/office-gw/files/export?fileGuid=%s&type=%s",
	Query:  "https://shimo.im/lizard-api/office-gw/files/export/progress?taskId=%s",
}

// Shared types
type FileInfo struct {
	Path   string
	Id     string `json:"guid"`
	Title  string `json:"name"`
	Type   string `json:"type"`
	TaskId string
}

type FileList []FileInfo

type DirInfo struct {
	FileInfo
	Dirs  *DirList
	Files *FileList
}

type DirList []DirInfo

func (tree DirInfo) String() string {
	str := fmt.Sprintf("type: %s id: %s title: %s path: %s &dirs: %p &files: %p", tree.Type, tree.Id, tree.Title, tree.Path, tree.Dirs, tree.Files)
	return str
}

func (file FileInfo) String() string {
	str := fmt.Sprintf("type: %s id: %s title: %s path: %s", file.Type, file.Id, file.Title, file.Path)
	return str
}

type FileResponse []FileInfo

type ExportResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	TaskId  string `json:"taskId,omitempty"`
}

type TaskResponse struct {
	Status int `json:"status"`
	Code   int `json:"code"`
	Data   struct {
		Progress    int    `json:"progress"`
		DownloadUrl string `json:"downloadUrl"`
		FileSize    int    `json:"fileSize"`
		CostTime    int    `json:"costTime"`
	} `json:"data,omitempty"`
}

type Logger interface {
	Log(message string)
}

// DefaultLogger is a simple logger that prints messages to the console
type DefaultLogger struct{}

// Log prints the message to the console
func (l *DefaultLogger) Log(message string) {
	fmt.Println(message)
}

// Define the UserCfg struct
type UserCfg struct {
	ExportType  string
	ShimoSid    string
	RootPath    string
	RemoveBlank bool
}

// Define the HttpStrategy struct
type HttpStrategy struct {
	SleepTime time.Duration
	Retry10   int
	Retry2    int
	Retry20   int
}

// Update the Shimo struct to include UserCfg, HttpStrategy, Logger, and fileCount
type Shimo struct {
	UserCfg      UserCfg
	HttpStrategy HttpStrategy
	Logger       Logger
	FileCount    int
	PauseChan    chan struct{}
}

// Update the NewShimo function to initialize FileCount
func NewShimo(userCfg UserCfg, httpStrategy HttpStrategy, logger Logger) *Shimo {
	return &Shimo{
		UserCfg:      userCfg,
		HttpStrategy: httpStrategy,
		Logger:       logger,
		FileCount:    0,
		PauseChan:    make(chan struct{}),
	}
}

// Update the NewDefaultShimo function to initialize FileCount
func NewDefaultShimo() *Shimo {
	return &Shimo{
		UserCfg: UserCfg{
			ExportType:  "md",
			ShimoSid:    "",
			RootPath:    "./download",
			RemoveBlank: false,
		},
		HttpStrategy: HttpStrategy{
			SleepTime: 10 * time.Second,
			Retry10:   10,
			Retry2:    2,
			Retry20:   20,
		},
		Logger:    &DefaultLogger{},
		FileCount: 0,
		PauseChan: make(chan struct{}),
	}
}

// Option is a function that configures a Shimo instance
type Option func(*Shimo)

// WithExportType sets the ExportType for a Shimo instance
func WithExportType(exportType string) Option {
	return func(s *Shimo) {
		s.UserCfg.ExportType = exportType
	}
}

// WithShimoSid sets the ShimoSid for a Shimo instance
func WithShimoSid(shimoSid string) Option {
	return func(s *Shimo) {
		s.UserCfg.ShimoSid = shimoSid
	}
}

func WithRootPath(rootPath string) Option {
	return func(s *Shimo) {
		s.UserCfg.RootPath = rootPath
	}
}

func WithRemoveBlank(removeBlank bool) Option {
	return func(s *Shimo) {
		s.UserCfg.RemoveBlank = removeBlank
	}
}

// WithLogger sets the Logger for a Shimo instance
func WithLogger(logger Logger) Option {
	return func(s *Shimo) {
		s.Logger = logger
	}
}

// Initialize a new Shimo instance with options
func NewShimoWithOptions(options ...Option) *Shimo {
	shimo := NewDefaultShimo()
	for _, option := range options {
		option(shimo)
	}

	return shimo
}

// Internal function to make HTTP requests
func httpRequest(uri string, retry int, shimoSid string, sleepTime time.Duration) ([]byte, error) {
	defaultstr := []byte("http request error occur")

	// 创建一个http.Client
	client := &http.Client{}
	//fmt.Println(uri)

	// 创建一个http.Request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return defaultstr, err
	}

	req.Header.Set("referer", "https://shimo.im/desktop")

	// 创建一个Cookie
	cookie := &http.Cookie{
		Name:  "shimo_sid",
		Value: shimoSid,
	}

	// 将Cookie添加到Request中
	req.AddCookie(cookie)

	// 使用Client发送Request
	resp, err := client.Do(req)
	if nil != err {
		return defaultstr, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		if retry > 0 {
			fmt.Println("429 too many requests, retry: ", retry, "...", uri)
			time.Sleep(sleepTime)
			return httpRequest(uri, retry-1, shimoSid, sleepTime)
		}
	}

	if resp.StatusCode != 200 {
		return defaultstr, errors.New("status error: " + resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if nil != err {
		return defaultstr, err
	}

	return body, nil
}

// Internal method to fetch directory information
func (s *Shimo) fetchDirectoryInfo(path string, id string, d *DirList, f *FileList) {
	uri := smUrl.Root
	if id != "" {
		uri = fmt.Sprintf(smUrl.List, id)
	}

	b, err := httpRequest(uri, s.HttpStrategy.Retry2, s.UserCfg.ShimoSid, s.HttpStrategy.SleepTime)
	if err != nil {
		panic(err)
	}

	var result FileResponse
	if err := json.Unmarshal(b, &result); err != nil {
		panic(err)
	}

	for _, fileInfo := range result {
		switch fileInfo.Type {
		case "folder":
			*d = append(*d, createDirInfo(path, fileInfo))
		case "newdoc":
			*f = append(*f, createFileInfo(path, fileInfo))
		}
	}
}

// createDirInfo 创建目录信息
func createDirInfo(path string, fileInfo FileInfo) DirInfo {
	return DirInfo{
		FileInfo: FileInfo{
			Path:  path + "/" + fileInfo.Title,
			Id:    fileInfo.Id,
			Title: fileInfo.Title,
			Type:  fileInfo.Type,
		},
		Dirs:  nil,
		Files: nil,
	}
}

// createFileInfo 创建文件信息
func createFileInfo(path string, fileInfo FileInfo) FileInfo {
	return FileInfo{
		Path:  path + "/" + fileInfo.Title,
		Id:    fileInfo.Id,
		Title: fileInfo.Title,
		Type:  fileInfo.Type,
	}
}

// title重复时,累计添加(1)
func duplicateTitle(path string) string {
	_, err := os.Stat(path)
	if err == nil {
		path = path + "(1)"
		path = duplicateTitle(path)
	} else if os.IsNotExist(err) {

	} else {
		panic(errors.New("duplicateTitle error "))
	}
	return path
}

func RemoveBlank(path string) string {
	fmt.Println("【移除空格】: ", path)
	return strings.ReplaceAll(path, " ", "")
}

// HttpExport 获取导出文件,默认按照md格式
func (s *Shimo) HttpExport(id string) (tid string) {
	s.Logger.Log(fmt.Sprintf("  [获取导出ID]: %s", id))
	uri := fmt.Sprintf(smUrl.Export, id, s.UserCfg.ExportType)

	b, err := httpRequest(uri, s.HttpStrategy.Retry10, s.UserCfg.ShimoSid, s.HttpStrategy.SleepTime)
	var result ExportResponse
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	err = json.Unmarshal(b, &result)
	if err != nil {
		fmt.Println(err)
	}

	if result.TaskId == "" {
		panic(errors.New("TaskId empty"))
	}

	s.Logger.Log(fmt.Sprintf("    [导出ID]: %+v", result.TaskId))
	return result.TaskId
}

// HttpLinkQuery 查询石墨导出进度
func (s *Shimo) HttpLinkQuery(tid string) string {
	uri := fmt.Sprintf(smUrl.Query, tid)
	b, err := httpRequest(uri, s.HttpStrategy.Retry20, s.UserCfg.ShimoSid, s.HttpStrategy.SleepTime)
	var result TaskResponse
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	err = json.Unmarshal(b, &result)
	if err != nil {
		fmt.Println(err)
	}

	s.Logger.Log(fmt.Sprintf("    [当前导出进度]: %+v", result.Data.Progress))

	if result.Status != 0 || result.Data.DownloadUrl == "" {
		time.Sleep(2 * time.Second)
		// 针对结果做循环调用，查询是否完成
		return s.HttpLinkQuery(tid)
	} else {
		s.Logger.Log(fmt.Sprintf("    [下载地址]: %+v", result.Data.DownloadUrl))
	}

	return result.Data.DownloadUrl
}

// HttpDownload 下载文件
func (s *Shimo) HttpDownload(uri string) []byte {
	fmt.Println("[httpDownload]: start download:", uri)
	b, err := httpRequest(uri, s.HttpStrategy.Retry2, s.UserCfg.ShimoSid, s.HttpStrategy.SleepTime)

	if err != nil {
		fmt.Println(err)
	}

	return b
}

// download a file to disk
func (s *Shimo) diskDownload(f FileInfo) {
	dl := s.HttpLinkQuery(f.TaskId)
	s.Logger.Log(fmt.Sprintf("  [开始下载任务]: %+v", f.TaskId))
	b := s.HttpDownload(dl)

	dir := filepath.Dir(f.Path)

	if s.UserCfg.RemoveBlank {
		dir = RemoveBlank(dir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}

	path := duplicateTitle(f.Path)
	if s.UserCfg.RemoveBlank {
		path = RemoveBlank(path)
	}

	file, err := os.Create(path + "." + s.UserCfg.ExportType)
	if err != nil {
		fmt.Println(err)
	} else {
		s.Logger.Log(fmt.Sprintf("    [创建文件成功]: %s", path+"."+s.UserCfg.ExportType))
	}
	defer file.Close()

	file.Write(b)
	s.Logger.Log(fmt.Sprintf("    [文件写入成功]: %s", path+"."+s.UserCfg.ExportType))
	s.Logger.Log(fmt.Sprintf("    [下载成功]: %s", path+"."+s.UserCfg.ExportType))
}

// CheckShimoSid 检查shimoSid是否有效
func (s *Shimo) CheckShimoSid() error {
	uri := smUrl.Root
	resp, err := httpRequest(uri, s.HttpStrategy.Retry2, s.UserCfg.ShimoSid, s.HttpStrategy.SleepTime)
	if err != nil {
		return err
	}

	// Assuming a 200 status code means the shimoSid is valid
	if string(resp) == "http request error occur" {
		return errors.New("invalid shimoSid")
	}

	return nil
}

// Internal function to build the structure tree
func (s *Shimo) BuildStructureTree(tree *DirInfo) {
	dirs := &DirList{}
	files := &FileList{}

	s.fetchDirectoryInfo(tree.Path, tree.Id, dirs, files)

	tree.Files = files
	tree.Dirs = dirs

	for _, file := range *files {
		s.Logger.Log(fmt.Sprintf("- [文件]: %s", file.Path))
		s.FileCount++
	}

	for i := range *dirs {
		s.BuildStructureTree(&(*dirs)[i])
	}
}

// TraverseTree 遍历文件结构
func (s *Shimo) TraverseTree(tree *DirInfo) {
	node := *tree

	if node.Files != nil {
		fl := *(node.Files)
		// Check for pause signal before starting a new download
		select {
		case <-s.PauseChan:
			s.Logger.Log("------> [下载暂停] <------")
			<-s.PauseChan // Wait for resume signal
			s.Logger.Log("------> [下载继续] <------")
		default:
			// No pause signal, continue with download
		}

		for i := range fl {
			s.Logger.Log(fmt.Sprintf("[开始下载]: %s 类型: %s", fl[i].Title, fl[i].Type))

			tid := s.HttpExport(fl[i].Id)
			(*(node.Files))[i].TaskId = tid
			s.diskDownload(fl[i])
			s.Logger.Log(fmt.Sprintf("[下载完成]: %s 地址：%s\n", fl[i].Title, fl[i].Path))
			s.Logger.Log(("-------------------------------"))
		}
	}

	if *(node.Dirs) == nil {
		s.Logger.Log(fmt.Sprintf(node.Id, "dir nil"))
		return
	}

	// 深度遍历
	dl := *(node.Dirs)
	for i := range dl {
		s.TraverseTree(&dl[i])
	}
}
