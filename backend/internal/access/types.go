package access

type ResourceSlug string
type ActionCode string

const (
	Read   ActionCode = "read"
	Write  ActionCode = "write"
	Delete ActionCode = "delete"
	All    ActionCode = "*"
)

var AllActions = []ActionCode{Read, Write, Delete}

type Permission struct {
	Resource ResourceSlug `json:"resource"`
	Action   ActionCode   `json:"action"`
}

func (p Permission) Key() string {
	return string(p.Resource) + ":" + string(p.Action)
}

// type ResourceGroups string

// const (
// 	GroupSystem ResourceGroups = "system"
// 	GroupAdmin  ResourceGroups = "admin"
// 	GroupLogs   ResourceGroups = "logs"
// 	GroupOps    ResourceGroups = "operations"
// )
