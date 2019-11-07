package gpool

// Task task interface
type Task interface {
	Run()
}

// TaskFunc task function define
type TaskFunc func()

// Run implement Task interface
func (sf TaskFunc) Run() {
	sf()
}
