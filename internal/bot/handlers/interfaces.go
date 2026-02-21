package handlers

type UserState int

const (
	StateNone UserState = iota

	StateWaitingForTask
	StateWaitingForSubtask
	StateWaitingForFiles
	StateWaitingForComment

	StateWaitingForWeeklyReport
	StateWaitingForCourseSelection
	StateWaitingForCuratorSelection

	StateWaitingForReminderText

	StateWaitingForBotName
	StateWaitingForWelcomeText
	StateWaitingForButtonKey
	StateWaitingForButtonNewText
	StateWaitingForReportTime
	StateWaitingForReminderTime
	StateWaitingForActivityTime
	StateWaitingForCuratorReportTime
	StateWaitingForReportDays

	StateWaitingForMessageCategory
	StateWaitingForMessageKey
	StateWaitingForMessageText
	StateWaitingForMessageImage

	StateWaitingForNewNickname
	StateWaitingForCourseID
	StateWaitingForCourseName

	StateWaitingForInputUserStats
	StateWaitingForDeleteInput
	StateWaitingForDeleteConfirm

	StateWaitingForTransferSource
	StateWaitingForTransferTarget
	StateWaitingForTransferConfirm

	StateCMSWaitingBotName
	StateCMSWaitingGreeting
	StateCMSWaitingButtons
	StateCMSWaitingMessages
	StateCMSWaitingReportTime
	StateCMSWaitingReminderTime
	StateCMSWaitingCheckTime
	StateCMSWaitingCuratorReportTime
	StateCMSWaitingWeeklyDays
)

type StateContext interface {
	SetState(userID int64, state UserState)
	GetState(userID int64) UserState
	ClearState(userID int64)

	SetData(userID int64, key string, value interface{})
	GetData(userID int64, key string) interface{}
	ClearData(userID int64)
}
