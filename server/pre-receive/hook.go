package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	EmptyRef        = "0000000000000000000000000000000000000000"
	ColorPurple     = "\033[35m"
	ColorRed        = "\033[31m"
	ColorRedBold    = "\033[1;31m"
	ColorYellow     = "\033[33m"
	ColorYellowBold = "\033[1;33m"
	ColorGreen      = "\033[32m"
	ColorGreenBold  = "\033[1;32m"
	ColorBlue       = "\033[34m"
	ColorBlueBold   = "\033[1;34m"
	ColorEnd        = "\033[0m"
)

type FileType string

const (
	FileTypePHP FileType = ".php"
	FileTypeJS  FileType = ".js"
	FileTypeGO  FileType = ".go"
)

type GitProtocol string

const (
	GitProtocolHttp GitProtocol = "http"
	GitProtocolSSH  GitProtocol = "ssh"
	GitProtocolWEB  GitProtocol = "web"
)

var codeExemptionRegexp = regexp.MustCompile(`\[A\]([0-9]+)\[/A\]`)

type Hook struct {
	Conf *Conf
	// 当前项目名称
	Repos string
	// 当前项目所属的命名空间
	NameSpace string
	// 临时目录
	TempDir     string
	GitProtocol GitProtocol
	// 是否合并请求
	IsMergeRequest bool
	OldRef         string
	NewRef         string
	Ref            string
	// 最新的提交
	NewObject *object.Commit
}

type CommitLog struct {
	// 提交人邮箱
	Author string
	// 上次提交
	OldRef string
	// 最新提交
	NewRef string
	// 提交路径: 如refs/head/develop
	Ref string
	// 版本库所属命名空间
	Namespace string
	// 版本库名
	Repos string
	// 提交中携带的jira号
	JiraIds []string
	// 文件变更列表
	FileStats []object.FileStat
	// 提交信息
	Message string
}

func (c *CommitLog) String() string {
	m, _ := json.MarshalIndent(c, "", "    ")
	return string(m)
}

// FindCodeExemption code
func (h *Hook) FindCodeExemption(message string) string {
	ret := codeExemptionRegexp.FindStringSubmatch(message)
	if len(ret) != 2 {
		return ""
	}
	return ret[1]
}

func (h *Hook) IsIgnoreNamespace() bool {
	for _, value := range h.Conf.IgnoreNamespace {
		if value == h.NameSpace {
			return true
		}
	}
	return false
}

func (h *Hook) IsIgnoreRepos() bool {
	for _, value := range h.Conf.IgnoreRepos {
		if value == h.Repos {
			return true
		}
	}
	return false
}

func (h *Hook) IsSuperAccount(email string) bool {
	for _, superAccount := range h.Conf.SuperAccount {
		if superAccount == email {
			return true
		}
	}
	return false
}

// IsProtectBranch
// ref = refs/heads/$branch
func (h *Hook) IsProtectBranch(ref string) bool {
	for _, value := range h.Conf.ProtectBranch {
		r := fmt.Sprintf("refs/heads/%s", value)
		if r == ref {
			return true
		}
	}
	return false
}

func (h *Hook) GetJiraID(message string) []string {
	reg := regexp.MustCompile(h.Conf.RequireJiraIDRexp)
	ret := reg.FindAllStringSubmatch(message, 1)
	if len(ret) > 0 && len(ret[0]) > 0 {
		m := make(map[string]bool)
		for _, j := range ret[0] {
			if j == "" {
				continue
			}
			m[j] = true
		}
		var r []string
		for k := range m {
			if k == "" {
				continue
			}
			r = append(r, k)
		}
		return r
	}
	return []string{}
}

func (h *Hook) CodeExemptionCheck(message string) bool {
	code := h.FindCodeExemption(message)
	if code != "" {
		codeFile := fmt.Sprintf("%s/%s", strings.TrimRight(h.Conf.CodeExemptionDir, "/"), code)
		_, err := os.Lstat(codeFile)
		if !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

// ParseEnv
// see https://docs.gitlab.com/ee/administration/server_hooks.html
func (h *Hook) parseEnv() {
	p := strings.Split(os.Getenv("GL_PROJECT_PATH"), "/")
	h.Repos = p[1]
	h.NameSpace = p[0]

	proto := os.Getenv("GL_PROTOCOL")
	switch proto {
	case "ssh":
		h.GitProtocol = GitProtocolSSH
	case "http":
		h.GitProtocol = GitProtocolHttp
	case "web":
		h.GitProtocol = GitProtocolWEB
	}
}

func (h *Hook) LoadConfig() error {
	viper.SetConfigName("pre-receive.yaml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc")

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	viper.AddConfigPath(dir)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return viper.Unmarshal(&h.Conf)
}

func (h *Hook) Info(color string, format string, param ...interface{}) {
	message := fmt.Sprintf(format, param...)
	if h.GitProtocol == GitProtocolWEB {
		fmt.Printf("GL-HOOK-ERR: %s", message)
		return
	}
	fmt.Printf("%s %s %s\n", color, message, ColorEnd)
}

func (h *Hook) InfoHeader(oldRef, newRef, ref string) {
	// disable gitlab ui print header
	//if h.GitProtocol == GitProtocolWEB {
	//	return
	//}
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\bcode exemption: insert \"[A]code[/A]\" into commit message")
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b    repository: %s", h.Repos)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b     namespace: %s", h.NameSpace)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b       old_ref: %s", oldRef)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b       new_ref: %s", newRef)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b           ref: %s", ref)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b           protocol: %s", h.GitProtocol)
	// h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b        env: %+v", os.Environ())
}

func (h *Hook) ParseDiffChangeStats(oldRef, newRef string) ([]object.FileStat, error) {
	var out []byte
	var err error
	var f []object.FileStat
	if oldRef == EmptyRef {
		out, err = sh.Command("git", "--no-pager", "show", "--numstat", "--stat", newRef).Output()
	} else {
		out, err = sh.Command("git", "--no-pager", "diff", "--numstat", "--stat", h.OldRef+"..."+h.NewRef).Output()
	}

	if err != nil {
		return f, err
	}

	// out:
	// 5	0	README.md
	// 1	2	a/b/c/m.go
	// README.md  | 5 +++++
	// a/b/c/m.go | 3 +--
	// 2 files changed, 6 insertions(+), 2 deletions(-)
	arr := strings.Split(string(out), "\n")
	for _, line := range arr {
		if !strings.Contains(line, "\t") {
			continue
		}
		statArr := strings.Split(strings.TrimSpace(line), "\t")
		if len(statArr) != 3 {
			continue
		}
		addition, _ := strconv.ParseInt(statArr[0], 10, 64)
		deletion, _ := strconv.ParseInt(statArr[1], 10, 64)
		stat := object.FileStat{
			Name:     statArr[2],
			Addition: int(addition),
			Deletion: int(deletion),
		}
		f = append(f, stat)
	}
	return f, nil
}

func (h *Hook) CommitLog() (*CommitLog, error) {
	if h.IsMergeRequest {
		return nil, nil
	}

	if h.NewRef == EmptyRef {
		return nil, errors.New("commit object is nil")
	}

	log := &CommitLog{
		Author:    h.NewObject.Author.Email,
		OldRef:    h.OldRef,
		NewRef:    h.NewRef,
		Ref:       h.Ref,
		Namespace: h.NameSpace,
		Repos:     h.Repos,
		JiraIds:   h.GetJiraID(h.NewObject.Message),
		Message:   h.NewObject.Message,
	}

	stats, err := h.ParseDiffChangeStats(h.OldRef, h.NewRef)
	if err != nil {
		return log, err
	}
	log.FileStats = stats

	if !h.Conf.CommitLogHook.Http.Enable {
		return log, nil
	}

	client := &http.Client{
		Timeout: time.Second * 3,
	}

	request, err := http.NewRequest("POST", h.Conf.CommitLogHook.Http.ReceiveURL, strings.NewReader(log.String()))
	if err != nil {
		return log, err
	}

	for k, v := range h.Conf.CommitLogHook.Http.Header {
		request.Header.Set(k, v)
	}

	resp, err := client.Do(request)
	if err != nil {
		return log, err
	}

	if resp.StatusCode != 200 {
		return log, fmt.Errorf("http response code was: %d", resp.StatusCode)
	}

	return log, nil
}

// Run
// return value: when success 0, otherwise > 0
func (h *Hook) Run(oldRev, newRev, ref string) int {
	h.OldRef = oldRev
	h.NewRef = newRev
	h.Ref = ref

	h.parseEnv()
	err := h.CreateTempDir()
	if err != nil {
		h.Info(ColorRedBold, "create temp dir err: %s", err)
		return 1
	}

	// load config
	err = h.LoadConfig()
	if err != nil {
		h.Info(ColorRedBold, "load config err: %s", err)
		return 1
	}

	if h.Conf.ClearCache {
		defer func() {
			h.ClearTemp()
		}()
	}

	h.InfoHeader(oldRev, newRev, ref)

	// 控制台分支创建
	if oldRev == EmptyRef && newRev != EmptyRef && h.GitProtocol == GitProtocolWEB {
		return 0
	}

	// project path
	projectPath, err := os.Getwd()
	if err != nil {
		h.Info(ColorRedBold, "get project path err: %s", err)
		return 1
	}

	// open git local
	r, err := git.PlainOpen(projectPath)
	if err != nil {
		h.Info(ColorRedBold, "open repos err: %s", err)
		return 1
	}

	obj, err := r.CommitObject(plumbing.NewHash(newRev))
	if err != nil {
		h.Info(ColorRedBold, "get object(%s) err: %s", newRev, err)
		return 0
	}

	h.NewObject = obj

	// check email format
	isEmailValid := false
	emailSuf := strings.Split(obj.Author.Email, "@")[1]
	for _, allowEmail := range h.Conf.AllowEmail {
		if allowEmail == emailSuf {
			isEmailValid = true
			break
		}
	}

	if !isEmailValid {
		h.Info(ColorRedBold, "git config.email was not allowed : %s, require: %+v, command: git config user.email $email", obj.Author.Email, h.Conf.AllowEmail)
		return 1
	}

	// super account
	if h.IsSuperAccount(obj.Author.Email) {
		h.Info(ColorGreenBold, "Hey, you commit with a super account!")
		return 0
	}

	if h.IsIgnoreNamespace() {
		h.Info(ColorGreenBold, "namespace %s was ignored.", h.NameSpace)
		return 0
	}

	if h.IsIgnoreRepos() {
		h.Info(ColorGreenBold, "repos %s was ignored.", h.Repos)
		return 0
	}

	// code exemption
	if h.CodeExemptionCheck(obj.Message) {
		h.Info(ColorGreenBold, "congratulations, code exemption triggered!")
		return 0
	}

	// merge request
	keywords := []string{"合并分支", "Merge"}
	for _, key := range keywords {
		if strings.Contains(obj.Message, key) && h.GitProtocol == GitProtocolWEB {
			h.IsMergeRequest = true
			return 0
		}
	}

	if h.IsProtectBranch(ref) {
		h.Info(ColorRedBold, "this branch was protected, can't push directly!")
		return 1
	}

	// jiraID
	if h.Conf.RequireJiraIDRexp != "" {
		jiraIDArr := h.GetJiraID(obj.Message)
		if len(jiraIDArr) <= 0 {
			h.Info(ColorRedBold, "commit message must contain at lease one jira ID, rule: %s, use git commit --amend", h.Conf.RequireJiraIDRexp)
			h.Info(ColorBlue, "message: %s", obj.Message)
			return 1
		}
	}

	// code exemption
	if h.CodeExemptionCheck(obj.Message) {
		h.Info(ColorGreenBold, "congratulations, code exemption triggered!")
		return 0
	}

	// base rule passed
	h.Info(ColorGreenBold, "base rule check passed!")

	if oldRev == EmptyRef || newRev == EmptyRef {
		h.Info(ColorGreenBold, "commit not changed!")
		return 0
	}

	// check file code format or static check
	oldObject, err := r.CommitObject(plumbing.NewHash(oldRev))
	if err != nil {
		h.Info(ColorRedBold, "get old object(%s) err: %s", oldRev, err)
		return 1
	}

	oldTree, err := oldObject.Tree()
	if err != nil {
		h.Info(ColorRedBold, "get old object tree(%s) err: %s", oldRev, err)
		return 1
	}

	newTree, err := obj.Tree()
	if err != nil {
		h.Info(ColorRedBold, "get new object tree(%s) err: %s", newRev, err)
		return 1
	}

	changes, err := oldTree.Diff(newTree)
	if err != nil {
		h.Info(ColorRedBold, "get new object changes(%s...%s) err: %s", oldRev, newRev, err)
		return 1
	}

	stat := make(map[FileType]int)
	for _, c := range changes {
		_, toFile, err := c.Files()
		// delete file or not regular file skip check
		if err != nil || toFile == nil || toFile.Mode != filemode.Regular {
			continue
		}

		// when is disabled style check, stop checkout file
		fileExt := strings.ToLower(path.Ext(toFile.Name))
		if fileExt == "" {
			continue
		}
		fileType := FileType(fileExt)
		ok, err := h.StyleCheckConfCheck(fileType)
		if err != nil {
			h.Info(ColorRedBold, "style check conf was detected some err: %s", err)
			return 1
		}

		// style check was disabled, skip checkout file
		if !ok {
			continue
		}

		_, err = h.CreateTempFile(fileType, c.To.Name, toFile)
		if err != nil {
			h.Info(ColorRedBold, "get new object changes(%s...%s) err: %s", oldRev, newRev, err)
			return 1
		}

		//h.Info(ColorYellowBold, "create temp file: %s", tempFile)

		_, ok = stat[fileType]
		if !ok {
			stat[fileType] = 0
		}
		stat[fileType]++
	}

	for fileType := range stat {
		switch fileType {
		case FileTypePHP:
			checkRet := h.PHPStyleCheck()
			if checkRet > 0 {
				return checkRet
			}
		case FileTypeJS:
			checkRet := h.JSStyleCheck()
			if checkRet > 0 {
				return checkRet
			}
		case FileTypeGO:
			checkRet := h.GOStyleCheck()
			if checkRet > 0 {
				return checkRet
			}
		}
	}

	return 0
}

// ClearTemp delete temp file
func (h *Hook) ClearTemp() {
	if h.TempDir == "" {
		return
	}
	_ = os.RemoveAll(h.TempDir)
}

// StyleCheckConfCheck
// check conf if is ok
// return true was run style check
// error will stop the process then due to commit failed
func (h *Hook) StyleCheckConfCheck(t FileType) (bool, error) {
	switch t {
	case FileTypePHP:
		if !h.Conf.StyleCheck.PHP.Enable {
			return false, nil
		}
		if h.Conf.StyleCheck.PHP.PHPCS == "" {
			return false, errors.New("Conf.StyleCheck.PHP.PHPCS is empty")
		}
		_, err := os.Stat(h.Conf.StyleCheck.PHP.PHPCS)
		if err != nil {
			return false, fmt.Errorf("can't stat PHPCS: %s, err: %s", h.Conf.StyleCheck.PHP.PHPCS, err)
		}
	case FileTypeJS:
		if !h.Conf.StyleCheck.JS.Enable {
			return false, nil
		}
	case FileTypeGO:
		if !h.Conf.StyleCheck.GO.Enable {
			return false, nil
		}
		if h.Conf.StyleCheck.GO.GolangCiLint == "" {
			return false, errors.New("Conf.StyleCheck.GO.GolangCiLint is empty")
		}
		_, err := os.Stat(h.Conf.StyleCheck.GO.GolangCiLint)
		if err != nil {
			return false, fmt.Errorf("can't stat golangci-lint: %s, err: %s", h.Conf.StyleCheck.GO.GolangCiLint, err)
		}
	}
	return true, nil
}

func (h *Hook) CreateTempDir() error {
	tempDir, err := ioutil.TempDir("", "git-pre-receive-tmp")
	if err != nil {
		return err
	}
	h.TempDir = tempDir
	return nil
}

// e.g.:
// .PHP file save to /tmp/php/$git_full_path
// .JS file save to /tmp/js/$git_full_path
func (h *Hook) getRootPathByFileType(t FileType) string {
	return fmt.Sprintf("%s/%s", h.TempDir, string(t)[1:])
}

// CreateTempFile
// Notice: f.Name() was not the fullPath,
func (h *Hook) CreateTempFile(t FileType, fullPath string, f *object.File) (string, error) {
	tempFile := fmt.Sprintf("%s/%s", h.getRootPathByFileType(t), fullPath)
	p := filepath.Dir(tempFile)
	err := os.MkdirAll(p, 0700)
	if err != nil {
		return "", err
	}
	tf, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	content, err := f.Contents()
	if err != nil {
		return "", err
	}
	_, err = tf.WriteString(content)
	if err != nil {
		return "", err
	}
	_ = tf.Close()
	return tempFile, nil
}

// PHPStyleCheck check php code style
func (h *Hook) PHPStyleCheck() int {
	args := append(h.Conf.StyleCheck.PHP.PHPCSArgs, "-p", h.getRootPathByFileType(FileTypePHP))
	sess := sh.Command(h.Conf.StyleCheck.PHP.PHPCS, args...)
	sess.Stdout = os.Stdout
	err := sess.Run()
	if err != nil {
		h.Info(ColorRedBold, "PHP code style check was rejected!")
		return 1
	}
	return 0
}

// JSStyleCheck check php code style
func (h *Hook) JSStyleCheck() int {
	// @todo add js check
	return 0
}

// GOStyleCheck check php code style
func (h *Hook) GOStyleCheck() int {
	dir, err := os.Getwd()
	if err != nil {
		h.Info(ColorRedBold, "go code style check err: %s", err)
		return 1
	}
	defer func() {
		_ = os.Chdir(dir)
	}()
	// golangci-lint not support pass a absolute path
	_ = os.Chdir(h.getRootPathByFileType(FileTypeGO))

	args := h.Conf.StyleCheck.GO.GolangCiLintArgs
	sess := sh.Command(h.Conf.StyleCheck.GO.GolangCiLint, args...)
	sess.Stdout = os.Stdout
	err = sess.Run()
	if err != nil {
		h.Info(ColorRedBold, "go code style check was rejected!")
		return 1
	}
	return 0
}
