package apptypes

type Target struct {
	ContainerID string
	PreHook     string
	PostHook    string
	Paths       []string
}
