package session

import (
	"log"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type AliasFunction func(*Session, []string)

type Alias struct {
	Name    string
	Pattern string
	RE      *regexp.Regexp
	Fn      AliasFunction
	Enabled bool
	Count   uint
}

type AliasRegistry struct {
	Aliases map[string]Alias
}

func NewAliasRegistry() *AliasRegistry {
	return &AliasRegistry{Aliases: make(map[string]Alias)}
}

func (s *Session) AddAlias(alias Alias) {
	alias.RE = regexp.MustCompile(alias.Pattern)
	s.Aliases.Aliases[alias.Name] = alias
}

func (s *Session) RemoveAlias(name string) {
	if _, ok := s.Aliases.Aliases[name]; !ok {
		log.Printf("alias %s does not exist", name)
		return
	}
	delete(s.Aliases.Aliases, name)
}

func makeAliasRow(alias Alias) table.Row {
	return table.NewRow(table.RowData{
		"name":    alias.Name,
		"pattern": alias.Pattern,
		"enabled": alias.Enabled,
		"count":   alias.Count,
	})
}

func CmdAliases(s *Session, cmd string) {
	var rows []table.Row
	for _, alias := range s.Aliases.Aliases {
		rows = append(rows, makeAliasRow(alias))
	}

	t := table.New([]table.Column{
		table.NewColumn("name", "Name", 25).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("pattern", "Pattern", 30).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("enabled", "Enabled", 10).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center)),
		table.NewColumn("count", "Count", 20).WithStyle(lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("#8c8"))),
	}).
		WithRows(rows).
		BorderRounded()

	s.Output(t.View() + "\n")
}

// MatchAlias checks if the input matches any alias and executes it
func (s *Session) MatchAlias(input string) bool {
	input = strings.TrimSpace(input)
	
	for _, alias := range s.Aliases.Aliases {
		if !alias.Enabled {
			continue
		}
		
		if matches := alias.RE.FindStringSubmatch(input); matches != nil {
			alias.Count++
			s.Aliases.Aliases[alias.Name] = alias
			alias.Fn(s, matches)
			return true
		}
	}
	
	return false
}
