package report

import "time"

type Task struct {
	AdminChatID int64
	CourseID    string
	ReportType  string
}

type UserStat struct {
	UserID        int64
	Name          string
	Role          string
	HomeworkCount int
	FilesCount    int
	RegisteredAt  time.Time
}
