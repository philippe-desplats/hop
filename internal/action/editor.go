package action

// Editor is a code editor hop can open a project in. Bin is the launcher command
// written to [actions].editor; Name is the friendly label shown by hop setup.
type Editor struct {
	Name string
	Bin  string
}

// knownEditors is the auto-detection preference order used by hop setup, twin of
// knownAssistants. The first one found on PATH becomes the default editor.
var knownEditors = []Editor{
	{Name: "Cursor", Bin: "cursor"},
	{Name: "VS Code", Bin: "code"},
	{Name: "Zed", Bin: "zed"},
	{Name: "Sublime Text", Bin: "subl"},
	{Name: "Neovim", Bin: "nvim"},
	{Name: "Vim", Bin: "vim"},
}

// DetectEditors returns every known editor found on PATH, in preference order.
// It shares the package lookPath var with assistant detection, so tests stub once.
func DetectEditors() []Editor {
	var found []Editor
	for _, e := range knownEditors {
		if _, err := lookPath(e.Bin); err == nil {
			found = append(found, e)
		}
	}
	return found
}
