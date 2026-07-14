package model

import (
	"time"
)

type Attendance struct {
	ID               string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name             string    `gorm:"type:varchar(255);not null" json:"name"`
	Role             string    `gorm:"type:varchar(255)" json:"role"`
	AvatarInitials   string    `gorm:"type:varchar(10)" json:"avatarInitials"`
	Shift            string    `gorm:"type:varchar(255)" json:"shift"`
	ClockIn          string    `gorm:"type:varchar(50)" json:"clockIn"`
	ClockOut         string    `gorm:"type:varchar(50)" json:"clockOut"`
	Site             string    `gorm:"type:varchar(255)" json:"site"`
	Department       string    `gorm:"type:varchar(255)" json:"department"`
	Latitude         *float64  `gorm:"type:double precision" json:"latitude"`
	Longitude        *float64  `gorm:"type:double precision" json:"longitude"`
	Status           string    `gorm:"type:varchar(50)" json:"status"` // "Present", "Late", "Absent", "On Leave"
	HasOvertime      bool      `gorm:"type:boolean;default:false" json:"hasOvertime"`
	CorrectionReason string    `gorm:"type:text" json:"correctionReason"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
