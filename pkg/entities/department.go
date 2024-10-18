package entities

type DepartmentID int

type Department struct {
	ID         DepartmentID       `json:"id,omitempty"`
	Name       string             `json:"name,omitempty"`
	Permission PermissionRelation `json:"permissions,omitempty"`
}
