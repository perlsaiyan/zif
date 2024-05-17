package session

func (s *Session) Output(msg string) {
	s.Content += msg

	s.Sub <- UpdateMessage{Session: s.Name, Content: msg}
}
