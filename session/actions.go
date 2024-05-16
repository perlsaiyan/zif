package session

import (
	"regexp"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type ActionFunction func(*Session, ActionMatches)

type ActionMatches struct {
	ANSILine string
	Line     string
	Matches  []string
}

type Action struct {
	Name    string
	Pattern string
	Color   bool
	Enabled bool
	RE      *regexp.Regexp
	Fn      ActionFunction
	Count   uint
}

type ActionRegistry struct {
	Actions map[string]Action
}

func NewActionRegistry() *ActionRegistry {
	ar := ActionRegistry{Actions: make(map[string]Action)}

	// TODO: stick an sample action here for now
	action := Action{
		Name:    "RoomScanner",
		Pattern: "\x1b\\[1;35m",
		Color:   true,
		Enabled: true,
		Fn:      PossibleRoomScanner,
	}
	action.RE = regexp.MustCompile(action.Pattern)
	ar.Actions[action.Name] = action

	return &ar
}

func (s *Session) AddAction(action Action) {
	s.Actions.Actions[action.Name] = action
}

func (s *Session) RemoveAction(name string) {
	delete(s.Actions.Actions, name)
}

func makeActionsRow(action Action) table.Row {

	return table.NewRow(table.RowData{
		"name":    action.Name,
		"enabled": action.Enabled,
		"count":   action.Count,
	})
}

func CmdActions(s *Session, cmd string, h *SessionHandler) {
	var rows []table.Row
	for _, i := range s.Actions.Actions {
		rows = append(rows, makeActionsRow(i))
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("enabled", "Enabled", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("count", "Count", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.Output("Actions:\n" + t.View() + "\n")
}

func PossibleRoomScanner(s *Session, matches ActionMatches) {
	re_room_no_compass, _ := regexp.Compile(`^.* (\[ [ NSWEUD<>v^\|\(\)\[\]]* \] *$)`)
	re_room_compass, _ := regexp.Compile(`^.* \|`)
	//re_room_here, _ := regexp.Compile(`^Here +- `)
	//re_room_no_exits, _ := regexp.Compile(`^.* \[ No exits! \]`)

	room := false
	msg := "Potential Room"
	if re_room_compass.MatchString(matches.Line) {
		room = true
		msg += " with compass"
	}
	if re_room_no_compass.MatchString(matches.Line) {
		room = true
		msg += " without compass"
	}
	if room {
		s.Output(msg + "\n")
	}
}

func (s *Session) ActionParser(line []byte) {
	test := string(line)
	striptest := stripansi.Strip(test)

	for _, a := range s.Actions.Actions {

		if a.RE.MatchString(test) {
			a.Count += 1
			s.Actions.Actions[a.Name] = a
			m := ActionMatches{
				ANSILine: test,
				Line:     strings.TrimRight(striptest, "\r\n"),
				Matches:  a.RE.FindStringSubmatch(test),
			}
			a.Fn(s, m)
		}

	}

}
