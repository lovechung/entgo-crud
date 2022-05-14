package biz

import (
	"time"
)

type User struct {
	Id        int64
	Username  *string
	Password  *string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

type UserReply struct {
	Id       int64
	Username string
	Password string
}

type UserCarsReply struct {
	Id        int64
	Username  string
	CreatedAt string
	Cars      []*CarReply
}

type Car struct {
	Id           int64
	UserId       *int64
	Model        *string
	RegisteredAt *time.Time
}

type CarReply struct {
	Id           int64
	Model        string
	RegisteredAt string
}
