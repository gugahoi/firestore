package keys

type ContextKey string

const (
	ClientKey    ContextKey = "firestore-client"
	ProjectIDKey ContextKey = "project-id"
)
