// Package notify delivers native desktop notifications without a separate application runtime.
package notify

import "context"

type Notification struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Icon    string `json:"icon"`
}

type Sender interface {
	Send(context.Context, Notification, string) error
}

type Native struct{}
