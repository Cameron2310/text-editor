package editor

type Snapshot struct {
	Content []string
	Pos Position
}

func (snap *Snapshot) Restore() ([]string, Position) {
	return snap.Content, snap.Pos
}
