package editor

type Snapshot struct {
	Content []string
	Pos Position
}

// TODO: update restore to update config rather than return vals
func (snap *Snapshot) Restore() ([]string, Position) {
	return snap.Content, snap.Pos
}
