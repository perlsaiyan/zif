package session

func (s *Session) Output(msg string) {
	s.Content += msg
}
