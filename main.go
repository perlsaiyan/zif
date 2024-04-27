package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/perlsaiyan/zif/session"

	"github.com/mistakenelf/teacup/statusbar"
)

const useHighPerformanceRenderer = false

type textinputMsg struct {
	password_mode   bool
	toggle_password bool
}

type ZifModel struct {
	Name               string
	Input              textinput.Model
	LeftSideBar        viewport.Model
	LeftSideBarActive  bool
	RightSideBar       viewport.Model
	RightSideBarActive bool
	Viewport           viewport.Model
	SessionHandler     session.SessionHandler
	StatusBar          statusbar.Model
	Ready              bool
}

func (m ZifModel) Init() tea.Cmd {
	return tea.Batch(
		waitForActivity(m.SessionHandler.Sub), // wait for activity
		tea.SetWindowTitle("zif"),
	)
}

func (m ZifModel) View() string {
	if !m.Ready {
		return "\n  Initializing..."
	}
	//return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	var verts []string
	if m.LeftSideBarActive {
		verts = append(verts, m.LeftSideBar.View())
	}
	verts = append(verts, m.Viewport.View())
	if m.RightSideBarActive {
		verts = append(verts, m.RightSideBar.View())
	}

	return lipgloss.JoinVertical(lipgloss.Top, lipgloss.JoinHorizontal(lipgloss.Top, verts...), m.Input.View(), m.StatusBar.View())
}

func (m *ZifModel) ToggleSideBar(side string) {
	switch side {
	case "left":
		m.LeftSideBarActive = !m.LeftSideBarActive
		if m.LeftSideBarActive {
			m.Viewport.Width -= 25
		} else {
			m.Viewport.Width += 25
		}

	case "right":

		m.RightSideBarActive = !m.RightSideBarActive

		if m.RightSideBarActive {
			m.Viewport.Width -= 25
		} else {
			m.Viewport.Width += 25
		}
	}
}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return tea.Msg(<-sub)
	}
}

func (m ZifModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		//cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case session.SessionChangeMsg:

	case session.UpdateMessage:
		jump := m.Viewport.AtBottom()
		m.Viewport.SetContent(m.SessionHandler.ActiveSession().Content)
		if jump {
			m.Viewport.GotoBottom()
		}
		cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))

	case textinputMsg:
		if msg.toggle_password {
			if msg.password_mode {
				log.Printf("Turning on password mode\n")
				m.Input.EchoMode = textinput.EchoPassword

			} else {
				log.Printf("Turning off password mode\n")
				m.Input.EchoMode = textinput.EchoNormal
			}

			cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))
		}

	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" {
			return m, tea.Quit
		} else if k := msg.String(); k == "pgup" || k == "pgdown" || k == "end" || k == "home" {
			var viewcmd tea.Cmd
			switch k {
			case "end":
				m.Viewport.GotoBottom()
			case "home":
				m.Viewport.GotoTop()
			default:
				m.Viewport, viewcmd = m.Viewport.Update(msg)
			}

			cmds = append(cmds, viewcmd)
		} else if k := msg.String(); k == "f2" {
			m.ToggleSideBar("left")
		} else if k := msg.String(); k == "f3" {
			m.ToggleSideBar("right")
		} else if k := msg.String(); k == "enter" {
			m.Input.Placeholder = ""
			order := strings.TrimSpace(m.Input.Value())
			m.SessionHandler.HandleInput(order)
			m.Input.Reset()
		} else {
			var inputcmd tea.Cmd
			m.Input, inputcmd = m.Input.Update(msg)
			cmds = append(cmds, inputcmd)
		}

	case tea.WindowSizeMsg:
		//headerHeight := lipgloss.Height(m.headerView())
		footerHeight := 2
		//verticalMarginHeight := headerHeight + footerHeight
		verticalMarginHeight := footerHeight

		if !m.Ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			width := msg.Width
			if m.LeftSideBarActive {
				width -= 25
			}
			if m.RightSideBarActive {
				width -= 25
			}

			m.Viewport = viewport.New(width, msg.Height-verticalMarginHeight)
			m.Viewport.YPosition = 0
			m.Viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.Viewport.SetContent("Welcome to Zif, the Zero Insertion Force mud client.\n\n")
			m.Ready = true

			m.LeftSideBar = viewport.New(25, msg.Height-verticalMarginHeight)
			m.LeftSideBar.YPosition = 0
			m.LeftSideBar.HighPerformanceRendering = useHighPerformanceRenderer
			m.LeftSideBar.SetContent("LEFT")

			m.RightSideBar = viewport.New(25, msg.Height-verticalMarginHeight)
			m.RightSideBar.YPosition = 0
			m.RightSideBar.HighPerformanceRendering = useHighPerformanceRenderer
			m.RightSideBar.SetContent("RIGHT")

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			//m.viewport.YPosition = headerHeight + 1
		} else {
			width := msg.Width
			if m.LeftSideBarActive {
				width -= 25
			}
			if m.RightSideBarActive {
				width -= 25
			}

			m.Viewport.Width = width

			m.Viewport.Height = msg.Height - verticalMarginHeight
			m.LeftSideBar.Width = 25
			m.LeftSideBar.SetContent("LEFT")
			m.LeftSideBar.Height = msg.Height - verticalMarginHeight
			m.RightSideBar.Width = 25
			m.RightSideBar.SetContent("RIGHT")
			m.RightSideBar.Height = msg.Height - verticalMarginHeight

		}

		m.StatusBar.Height = 1
		m.StatusBar.SetSize(msg.Width)
		connected := func() string {
			if m.SessionHandler.ActiveSession().Connected {
				return "✓"
			} else {
				return "✗"
			}
		}
		m.StatusBar.SetContent(m.SessionHandler.ActiveSession().Name, "Not Connected", "100% Efficient", connected())
		m.Input.Width = msg.Width - 1

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.Viewport))
			if m.LeftSideBarActive {
				cmds = append(cmds, viewport.Sync(m.LeftSideBar))
			}

			if m.RightSideBarActive {
				cmds = append(cmds, viewport.Sync(m.RightSideBar))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func main() {

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	m := ZifModel{Input: textinput.New(), SessionHandler: session.NewHandler()}
	m.Input.Placeholder = "Welcome to Zif, type #HELP to get started"
	m.Input.Focus()
	m.Input.CharLimit = 156
	m.Input.Width = 20

	m.StatusBar = statusbar.New(statusbar.ColorConfig{
		Foreground: lipgloss.AdaptiveColor{Dark: "#ffffff", Light: "#ffffff"},
		Background: lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F25D94"},
	},
		statusbar.ColorConfig{
			Foreground: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},
			Background: lipgloss.AdaptiveColor{Light: "#3c3836", Dark: "#3c3836"},
		},
		statusbar.ColorConfig{
			Foreground: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},
			Background: lipgloss.AdaptiveColor{Light: "#A550DF", Dark: "#A550DF"},
		},
		statusbar.ColorConfig{
			Foreground: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},
			Background: lipgloss.AdaptiveColor{Light: "#6124DF", Dark: "#6124DF"},
		})

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	//go mudReader(m.sub, m.socket, &m)

	if _, err := p.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
