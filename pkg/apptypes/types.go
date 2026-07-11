package apptypes

type Target struct {
	ContainerID string
	Name        string
	PreHook     string
	PostHook    string
	Paths       []string
	Stop        bool
	Lock        bool
}
