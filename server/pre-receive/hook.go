package main

import (
	"errors"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
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

var codeExemptionRegexp = regexp.MustCompile(`\[A\]([0-9]+)\[/A\]`)

type Hook struct {
	Conf *Conf
	// 当前项目名称
	Repos string
	// 当前项目所属的命名空间
	NameSpace string
	// 临时目录
	TempDir string
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
		return ret[0]
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
	fmt.Printf("%s %s %s\n", color, message, ColorEnd)
}

func (h *Hook) InfoHeader(oldRef, newRef, ref string) {
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\bcode exemption: insert \"[A]code[/A]\" into commit message")
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b    repository: %s", h.Repos)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b     namespace: %s", h.NameSpace)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b       old_ref: %s", oldRef)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b       new_ref: %s", newRef)
	h.Info(ColorYellowBold, "\b\b\b\b\b\b\b\b\b        branch: %s", ref)
}

// Run
// return value: when success 0, otherwise > 0
func (h *Hook) Run(oldRev, newRev, ref string) int {
	h.parseEnv()
	err := h.CreateTempDir()
	if err != nil {
		h.Info(ColorRedBold, "create temp dir err: %s", err)
		return 1
	}

	defer func() {
		h.ClearTemp()
	}()

	// load config
	err = h.LoadConfig()
	if err != nil {
		h.Info(ColorRedBold, "load config err: %s", err)
		return 1
	}

	h.InfoHeader(oldRev, newRev, ref)

	if oldRev == EmptyRef || newRev == EmptyRef {
		h.Info(ColorGreenBold, "commit not changed!")
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
		h.Info(ColorRedBold, "get object err: %s", err)
		return 1
	}

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
		if strings.Contains(obj.Message, key) {
			return 0
		}
	}

	// jiraID
	if h.Conf.RequireJiraIDRexp != "" {
		jiraIDArr := h.GetJiraID(obj.Message)
		if len(jiraIDArr) <= 0 {
			h.Info(ColorRedBold, "commit message must contain at lease one jira ID, rule: %s, use git commit --amend", h.Conf.RequireJiraIDRexp)
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
		fileType := FileType(strings.ToLower(path.Ext(toFile.Name)))
		ok, err := h.StyleCheckConfCheck(fileType)
		if err != nil {
			h.Info(ColorRedBold, "style check conf was detected some err: %s", err)
			return 1
		}

		// style check was disabled, skip checkout file
		if !ok {
			continue
		}

		tempFile, err := h.CreateTempFile(fileType, c.To.Name, toFile)
		if err != nil {
			h.Info(ColorRedBold, "get new object changes(%s...%s) err: %s", oldRev, newRev, err)
			return 1
		}
		h.Info(ColorYellowBold, "create temp file: %s", tempFile)

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
	// @todo add go check
	return 0
}
