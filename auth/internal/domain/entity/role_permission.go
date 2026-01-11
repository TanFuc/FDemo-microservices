package entity

type RolePermission struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID       int64          `gorm:"not null;uniqueIndex:idx_role_permission" json:"roleId"`
	PermissionID int64          `gorm:"not null;uniqueIndex:idx_role_permission" json:"permissionId"`
	Conditions   map[string]any `gorm:"type:jsonb" json:"conditions,omitempty"`

	// Relations
	Role       Role       `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"-"`
	Permission Permission `gorm:"foreignKey:PermissionID;constraint:OnDelete:CASCADE" json:"permission,omitempty"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
