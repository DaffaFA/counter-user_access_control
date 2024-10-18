package entities

type PermissionID string

type PermissionRelation map[PermissionID]struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

type Permission struct {
	ID     PermissionID `json:"id"`
	Name   string       `json:"name"`
	Parent *Permission  `json:"parent_id"`
}
