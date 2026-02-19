package submission

import "time"

type Type string

const (
	TypeHomework Type = "homework"
	TypeNotes    Type = "notes"
)

type Submission struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	CuratorID     int64     `json:"curator_id"`
	Type          Type      `json:"type"`
	TaskNumber    string    `json:"task_number"`
	Comment       string    `json:"comment"`
	SubmittedAt   time.Time `json:"submitted_at"`
	FilePaths     []string  `json:"file_paths"`
	OriginalNames []string  `json:"original_names"`
}

type Subtask struct {
	Code string
	Name string
}

type FileDTO struct {
	FileID   string
	FileName string
	Size     int64
}

type InputDTO struct {
	UserID      int64
	CuratorID   int64
	CourseName  string
	CuratorName string
	StudentName string
	Type        Type
	TaskNumber  string
	Comment     string
	Files       []FileDTO
}
