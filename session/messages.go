package session

type UpdateMessage struct {
	Session string
	Content string
}

// The active session has changed
type SessionChangeMsg struct {
	ActiveSession *Session
}

// incoming mud content message
type MudOutputMsg struct {
	Session string
	Content string
}

// changes to the text input bar
type TextinputMsg struct {
	Session         string
	Password_mode   bool
	Toggle_password bool
}
