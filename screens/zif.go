package screens

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const useHighPerformanceRenderer = true

type ZifModel struct {
	Sub      chan tea.Msg
	Name     string
	viewport viewport.Model
	input    textinput.Model
	ready    bool
	content  string
	Active   bool
}

type UpdateMessage struct {
	Session string
	Content string
}

type TextinputMsg struct {
	Session        string
	PasswordMode   bool
	TogglePassword bool
}

type SessionActivationMsg struct {
	Active string
}

func (m ZifModel) Init() tea.Cmd {
	m.input = textinput.New()
	m.input.Placeholder = "Zif Session"
	m.input.Focus()
	return nil
}

func (m ZifModel) View() string {
	log.Printf("rendering %s, and ready = %s\n", m.Name, m.ready)
	if !m.ready {
		return "Initializing..."
	}
	//return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	return fmt.Sprintf("%s\n%s", m.viewport.View(), m.footerView())
}

func (m ZifModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case SessionActivationMsg:
		if msg.Active == m.Name {
			m.Active = true
			log.Printf("Self activating (%s)\n", m.Name)
		} else {
			m.Active = false
		}
		return m, nil

	case tea.KeyMsg:
		log.Printf("Session %s got %+v", m.Name, msg)

		if k := msg.String(); k == "pgup" || k == "pgdown" || k == "end" || k == "home" {
			var viewcmd tea.Cmd
			m.viewport, viewcmd = m.viewport.Update(msg)
			cmds = append(cmds, viewcmd)
		} else if k := msg.String(); k == "enter" {
			order := strings.TrimSpace(m.input.Value())
			if len(order) > 0 {
				if order[0] == '#' {
					if order == "#MSDP" {
						//msdp_vals := fmt.Sprintf("MSDP:\n %+v", spew.Sdump(m.msdp))
						msdp_vals := "MSDP TBD\n"
						m.Sub <- UpdateMessage{Content: msdp_vals, Session: m.Name}
					} else if order == "#PASSWORD" {
						m.Sub <- TextinputMsg{PasswordMode: false, TogglePassword: true, Session: m.Name}
					}

				} else {
					//m.socket.Write([]byte(m.input.Value() + "\n"))
				}
			} else {
				//m.socket.Write([]byte("\n"))
			}
			m.input.Reset()
		} else {
			var inputcmd tea.Cmd
			m.input, inputcmd = m.input.Update(msg)
			log.Printf("Generated inputcmd %+v", inputcmd)
			cmds = append(cmds, inputcmd)
		}

	case tea.WindowSizeMsg:

		//headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		//verticalMarginHeight := headerHeight + footerHeight
		verticalMarginHeight := footerHeight

		if !m.ready {
			log.Printf("WindowSizeMsg with state %v", m.ready)
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = 0
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewport.SetContent(m.content)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			//m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.SetContent(m.content)
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		// only if we're the active session
		if m.Active {
			if useHighPerformanceRenderer {
				// Render (or re-render) the whole viewport. Necessary both to
				// initialize the viewport and when the window is resized.
				//
				// This is needed for high-performance rendering only.
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		}

	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)

}

func (m ZifModel) footerView() string {
	return ">" + m.input.View()
	//return lipgloss.JoinHorizontal(lipgloss.Center, line)
}
