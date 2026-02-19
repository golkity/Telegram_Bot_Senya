package user

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrCourseNotFound    = errors.New("course not found")
)

type Role string

const (
	RoleStudent   Role = "student"
	RoleCurator   Role = "curator"
	RoleDeveloper Role = "developer"
	RoleAdmin     Role = "admin"
)

type Course struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	FirstName          string    `json:"first_name"`
	LastName           string    `json:"last_name"`
	Role               Role      `json:"role"`
	CourseID           *string   `json:"course_id"`
	CuratorID          *int64    `json:"curator_id"`
	AdminNotifications bool      `json:"admin_notifications"`
	RegisteredAt       time.Time `json:"registered_at"`
}

type DailyStat struct {
	HomeworkStatus  string
	NotesStatus     string
	TotalFilesToday int
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) CanSubmitHomework() bool {
	if u.Role == RoleDeveloper && (u.CuratorID == nil || *u.CuratorID == 0) {
		return false
	}
	return true
}
