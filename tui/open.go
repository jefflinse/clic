package tui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jefflinse/clic/provider"
)

// editorFinishedMsg reports the outcome of an external editor session opened to
// view the response body.
type editorFinishedMsg struct{ err error }

// openInEditor writes the current response body to a temp file and opens it in
// the user's editor, suspending the studio for the duration. It is read-only:
// edits are not read back into the request.
func (s *studio) openInEditor() tea.Cmd {
	if s.resp.result == nil {
		s.flash = "no response to open"
		return nil
	}
	name, args := editorCommand(resolveEditor())
	if name == "" {
		s.flash = "set $EDITOR to open externally"
		return nil
	}
	path, err := writeTempResponse(s.resp.result)
	if err != nil {
		s.flash = "could not write temp file"
		return nil
	}
	c := exec.Command(name, append(args, path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// resolveEditor returns the editor command line, preferring $VISUAL, then
// $EDITOR, then a platform default.
func resolveEditor() string {
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	if v := os.Getenv("EDITOR"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}

// editorCommand splits an editor setting (which may carry flags, e.g. "code -w")
// into its binary and leading arguments. An empty setting yields an empty name.
func editorCommand(editor string) (string, []string) {
	fields := strings.Fields(editor)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}

// writeTempResponse writes the response body to a temp file whose extension is
// derived from the content type, so editors apply the right syntax highlighting.
func writeTempResponse(res *provider.Result) (string, error) {
	f, err := os.CreateTemp("", "clic-response-*"+extForContentType(res.ContentType))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(res.Body); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func extForContentType(ct string) string {
	switch {
	case strings.Contains(ct, "json"):
		return ".json"
	case strings.Contains(ct, "html"):
		return ".html"
	case strings.Contains(ct, "xml"):
		return ".xml"
	default:
		return ".txt"
	}
}
