package controlplanepermission

import "time"

type ApplyParameters struct {
	// NOTE(muvaf): Only Permission field has json tag because the rest of the
	// fields are used for path parameters and they are not supposed to be
	// serialized.

	// AccountName is the name of the account to which the team belongs.
	AccountName string `json:"-"`

	// TeamID is the ID of the team to which the permission applies.
	TeamID string `json:"-"`

	// ControlPlaneName is the name of the control plane to which the permission
	// applies.
	ControlPlaneName string `json:"-"`

	// Permission is the permission to grant to the team for a control plane.
	// Valid values are "editor", "viewer", and "owner".
	Permission string `json:"permission"`
}

type GetParameters struct {
	// AccountName is the name of the account to which the team belongs.
	AccountName string `json:"-"`

	// TeamID is the ID of the team to which the permission applies.
	TeamID string `json:"-"`
}

type DeleteParameters struct {
	// AccountName is the name of the account to which the team belongs.
	AccountName string `json:"-"`

	// TeamID is the ID of the team to which the permission applies.
	TeamID string `json:"-"`

	// ControlPlaneName is the name of the control plane to which the permission
	// applies.
	ControlPlaneName string `json:"-"`
}

type GetResponse struct {
	Permissions []PermissionResponse `json:"permissions"`
	Size        int                  `json:"size"`
	Page        int                  `json:"page"`
	Count       int                  `json:"count"`
}

type PermissionResponse struct {
	ControlPlaneName string `json:"controlPlaneName"`

	ControlPlaneID string     `json:"controlPlaneId"`
	TeamID         string     `json:"teamId"`
	AccountID      uint       `json:"accountId"`
	Privilege      string     `json:"privilege"`
	CreatorID      uint       `json:"creatorId"`
	CreatedAt      *time.Time `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt"`
}
