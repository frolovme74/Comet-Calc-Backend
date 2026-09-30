package session

import "sync"

const creatorID uint = 1

type User struct {
	ID uint
}

var (
	once    sync.Once
	current *User
)

func CurrentUser() *User {
	once.Do(func() {
		current = &User{ID: creatorID}
	})
	return current
}
