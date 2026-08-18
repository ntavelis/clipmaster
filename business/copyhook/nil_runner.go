package copyhook

type nilRunner struct{}

func (r nilRunner) Trigger() {
	// do nothing
}
