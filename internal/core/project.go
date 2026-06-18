package core

// Project is one switchable directory.
type Project struct {
	Name     string `json:"name"`
	Path     string `json:"path"`     // absolute, symlink-resolved
	Category string `json:"category"` // first path segment under the matched root
}
