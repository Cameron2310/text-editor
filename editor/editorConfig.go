package editor

import "golang.org/x/sys/unix"

type EditorConfig struct {
	Rows int
	cols int
	Pos Position
	StateIdx int
    firstRowToView int
	firstColToView int
	Content []string
}


func NewEditorConfig(fd int, req uint) *EditorConfig {
	winConfig, err := unix.IoctlGetWinsize(fd, req)

	if err != nil {
		panic(err)
	}

	return &EditorConfig{
		Rows: int(winConfig.Row), cols: int(winConfig.Col), Pos: Position{X: 1, Y: 0}, StateIdx: 1, 
		firstRowToView: 0, firstColToView: 0, Content: make([]string, int(winConfig.Row)),
	}
}


func (config *EditorConfig) CreateSnapshot() *Snapshot {
	tmp := config.clone()
	return &Snapshot{Content: tmp.Content, Pos: tmp.Pos}
}


func (config *EditorConfig) clone() *EditorConfig {
	tmp := make([]string, len(config.Content))
	copy(tmp, config.Content)

	return &EditorConfig{
		Rows: config.Rows,
		cols: config.cols,
		Pos: Position{X: config.Pos.X, Y: config.Pos.Y},
		StateIdx: config.StateIdx,
		firstRowToView: config.firstRowToView,
		firstColToView: config.firstColToView,
		Content: tmp,
	}
}
