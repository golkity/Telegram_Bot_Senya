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
	NotesCount    int
	HomeworkCount int
	FilesCount    int
	RegisteredAt  time.Time
}

type SubmissionStat struct {
	Date        string
	StudentName string
	CuratorName string
	Type        string
	Status      string
}

type StrictReportData struct {
	ReportTitle      string
	GenerationDate   string
	CourseName       string
	CuratorName      string
	Period           string
	WeekType         string
	CourseFilter     string
	TotalUsers       int
	CourseStudents   int
	CourseDevs       int
	ActiveStudents   int
	ActiveDevs       int
	TotalSubmissions int
	HWSubmissions    int
	NotesSubmissions int
	DaysInPeriod     string
	ReportTypeStat   string
	AvgSubmissions   float64
	HWNotesRatio     string
}

type UserSubmissionsData struct {
	Username    string
	TotalCount  int
	Submissions []SubmissionDetail
}

type SubmissionDetail struct {
	Number       int
	DateTime     string
	Type         string
	TaskName     string
	FilesCount   int
	Comment      string
	Status       string
	SubmissionID int64
}

type WeeklyReportData struct {
	Date        string
	StudentName string
	CuratorName string
	Text        string
}

type StudentSheetRecord struct {
	CuratorName  string
	StudentName  string
	Username     string
	CourseName   string
	HWCount      int
	NotesCount   int
	RegisteredAt string
}
