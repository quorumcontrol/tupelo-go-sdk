package tag

// Key represents a tag key.
type Key struct {
	name string
}

// NewKey creates or retrieves a string key identified by name.
// Calling NewKey consequently with the same name returns the same key.
func NewKey(name string) (Key, error) {
	return Key{name: name}, nil
}

// Name returns the name of the key.
func (k Key) Name() string {
	return k.name
}

// Mutator modifies a tag map.
type Mutator interface {
	Mutate(t *Map) (*Map, error)
}

type Map struct {
	// m map[Key]tagContent
}
