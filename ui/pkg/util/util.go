// Package util provides utility functions for UI message handling.
package util

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"mvdan.cc/sh/v3/shell"
)

// PrettyPath replaces the actual home path with ~.
func PrettyPath(path string) string {
	homedir, err := os.UserHomeDir()
	if err != nil || homedir == "" {
		return path
	}
	if !strings.HasPrefix(path, homedir) {
		return path
	}
	return filepath.Join("~", strings.TrimPrefix(path, homedir))
}

// Capitalize returns text with first character title-cased.
func Capitalize(text string) string {
	return cases.Title(language.English, cases.Compact).String(text)
}

type Cursor interface {
	Cursor() *tea.Cursor
}

func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func ReportError(err error) tea.Cmd {
	return CmdHandler(NewErrorMsg(err))
}

type InfoType int

const (
	InfoTypeInfo InfoType = iota
	InfoTypeSuccess
	InfoTypeWarn
	InfoTypeError
	InfoTypeUpdate
)

func NewInfoMsg(info string) InfoMsg {
	return InfoMsg{
		Type: InfoTypeInfo,
		Msg:  info,
	}
}

func NewWarnMsg(warn string) InfoMsg {
	return InfoMsg{
		Type: InfoTypeWarn,
		Msg:  warn,
	}
}

func NewErrorMsg(err error) InfoMsg {
	return InfoMsg{
		Type: InfoTypeError,
		Msg:  err.Error(),
	}
}

func ReportInfo(info string) tea.Cmd {
	return CmdHandler(NewInfoMsg(info))
}

func ReportWarn(warn string) tea.Cmd {
	return CmdHandler(NewWarnMsg(warn))
}

type (
	InfoMsg struct {
		Type InfoType
		Msg  string
		TTL  time.Duration
	}
	ClearStatusMsg struct{}
)

// IsEmpty checks if the [InfoMsg] is empty.
func (m InfoMsg) IsEmpty() bool {
	var zero InfoMsg
	return m == zero
}

// ExecShell parses a shell command string and executes it with exec.Command.
// Uses shell.Fields for proper handling of shell syntax like quotes and
// arguments while preserving TTY handling for terminal editors.
func ExecShell(ctx context.Context, cmdStr string, callback tea.ExecCallback) tea.Cmd {
	fields, err := shell.Fields(cmdStr, nil)
	if err != nil {
		return ReportError(err)
	}
	if len(fields) == 0 {
		return ReportError(errors.New("empty command"))
	}

	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	return tea.ExecProcess(cmd, callback)
}
