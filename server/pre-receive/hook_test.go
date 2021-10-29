package main

import (
	"path"
	"regexp"
	"strings"
	"testing"
)

func TestFindCodeExemption(t *testing.T) {
	code := "193433"
	message := "aafdsfds[A]" + code + "[/A]afdsfasf"
	h := &Hook{}
	ret := h.FindCodeExemption(message)
	if ret != code {
		t.Errorf("code not found")
	}
}

func TestFileExt(t *testing.T) {
	f := "/a/b/a/p.go"
	ext := path.Ext(f)
	if ext != ".go" {
		t.Error("got file ext failed")
	}
}

func TestJiraID(t *testing.T) {
	reg := regexp.MustCompile("([a-zA-Z]+-[0-9]+)")
	ret := reg.FindAllStringSubmatch("asfas PA-1345 PRQ-1454", 1)
	if len(ret) > 0 && len(ret[0]) > 0 {
		t.Logf("%+v\n", strings.Join(ret[0], "|"))
		return
	}
	t.Errorf("not found, %+v\n", ret)
}
