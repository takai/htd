package model

import "time"

type Kind string
type Status string

const (
	KindInbox      Kind = "inbox"
	KindNextAction Kind = "next_action"
	KindProject    Kind = "project"
	KindWaitingFor Kind = "waiting_for"
	KindSomeday    Kind = "someday"
	KindTickler    Kind = "tickler"
)

const (
	StatusActive    Status = "active"
	StatusDone      Status = "done"
	StatusCanceled  Status = "canceled"
	StatusDiscarded Status = "discarded"
	StatusArchived  Status = "archived"
)

func ValidKinds() []Kind {
	return []Kind{KindInbox, KindNextAction, KindProject, KindWaitingFor, KindSomeday, KindTickler}
}

func ValidStatuses() []Status {
	return []Status{StatusActive, StatusDone, StatusCanceled, StatusDiscarded, StatusArchived}
}

func IsTerminal(s Status) bool {
	return s == StatusDone || s == StatusCanceled || s == StatusDiscarded || s == StatusArchived
}

func IsActive(s Status) bool {
	return s == StatusActive
}

type Item struct {
	ID         string    `yaml:"id"`
	Title      string    `yaml:"title"`
	Kind       Kind      `yaml:"kind"`
	Status     Status    `yaml:"status"`
	Project    string    `yaml:"project,omitempty"`
	CreatedAt  time.Time `yaml:"created_at"`
	UpdatedAt  time.Time `yaml:"updated_at"`
	DueAt      *time.Time `yaml:"due_at,omitempty"`
	DeferUntil *time.Time `yaml:"defer_until,omitempty"`
	ReviewAt   *time.Time `yaml:"review_at,omitempty"`
	Source     string    `yaml:"source,omitempty"`
	Tags       []string  `yaml:"tags,omitempty"`
}
