package filemanager

// File item that is compatible with bubble's List model
type FileItem struct {
	Name string
	Path string
}

func (i FileItem) Title() string       { return i.Name }
func (i FileItem) Description() string { return " " }
func (i FileItem) FilterValue() string { return i.Path }
