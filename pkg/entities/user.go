package entities

import "time"

type User struct {
	ID           int                `json:"id,omitempty"`
	FullName     string             `json:"full_name,omitempty"`
	Username     string             `json:"username,omitempty"`
	Password     string             `json:"password,omitempty"`
	DepartmentID *DepartmentID      `json:"department_id,omitempty"`
	Department   *Department        `json:"department,omitempty"`
	Permission   PermissionRelation `json:"permissions,omitempty"`
	ExpiredAt    *time.Time         `json:"expired_at,omitempty"`
	ActivatedAt  *time.Time         `json:"activated_at,omitempty"`
	CreatedAt    *time.Time         `json:"created_at,omitempty"`
	UpdatedAt    *time.Time         `json:"updated_at,omitempty"`
}

type UserPagination struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}
