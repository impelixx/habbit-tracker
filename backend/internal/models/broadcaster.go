package models

// TaskBroadcaster defines interface for broadcasting task updates
type TaskBroadcaster interface {
	BroadcastTaskUpdate(task *Task, action string)
}
