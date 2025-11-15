package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"plugin"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/perlsaiyan/zif/config"
	"github.com/perlsaiyan/zif/session"

	"github.com/mistakenelf/teacup/statusbar"
)

const useHighPerformanceRenderer = false
const RightSideBarSize = 50
const LeftSideBarSize = 50

type ZifModel struct {
	Name               string
	Config             *config.Config
	Plugins            []*plugin.Plugin
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
			m.Viewport.Width -= LeftSideBarSize
		} else {
			m.Viewport.Width += LeftSideBarSize
		}

	case "right":

		m.RightSideBarActive = !m.RightSideBarActive

		if m.RightSideBarActive {
			m.Viewport.Width -= RightSideBarSize
		} else {
			m.Viewport.Width += RightSideBarSize
		}
	}
}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		//log.Printf("Waiting for message on sub channel")
		msg := <-sub
		//log.Printf("Got %+v message", msg)
		return tea.Msg(msg)
		//return tea.Msg(<-sub)
	}
}

func (m ZifModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		//cmd  tea.Cmd
		cmds []tea.Cmd
	)

	//log.Printf("Receiving message: %s", msg)
	switch msg := msg.(type) {

	case session.SessionChangeMsg:
		log.Printf("Setting active session to %s (test: %s)", msg.ActiveSession.Name, m.SessionHandler.Active)
		m.SessionHandler.Active = msg.ActiveSession.Name

		m.StatusBar.FirstColumn = m.SessionHandler.Active
		if m.SessionHandler.ActiveSession().Connected {
			m.StatusBar.SecondColumn = m.SessionHandler.ActiveSession().Address
		} else {
			m.StatusBar.SecondColumn = "Not Connected"
		}
		m.StatusBar.ThirdColumn = fmt.Sprintf("%d/1000", m.Viewport.TotalLineCount())
		m.Viewport.SetContent(m.SessionHandler.ActiveSession().Content)
		m.Viewport.GotoBottom()
		cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))

	case session.UpdateMessage:
		m.StatusBar.FirstColumn = m.SessionHandler.Active
		if m.SessionHandler.ActiveSession().Connected {
			roomName := m.SessionHandler.ActiveSession().MSDP.GetString("ROOM_NAME")
			if len(roomName) > 0 {
				if k, ok := m.SessionHandler.Plugins.Plugins["kallisti"]; ok {
					tp, err := k.Plugin.Lookup("TravelProgress")
					if err == nil {
						m.StatusBar.SecondColumn = tp.(func(*session.Session) string)(m.SessionHandler.ActiveSession()) +
							" " + roomName
					} else {
						m.StatusBar.SecondColumn = roomName
					}
				} else {
					m.StatusBar.SecondColumn = roomName
				}
			} else {
				m.StatusBar.SecondColumn = m.SessionHandler.ActiveSession().Address
			}
		} else {
			m.StatusBar.SecondColumn = "Not Connected"
		}
		m.StatusBar.ThirdColumn = fmt.Sprintf("%d", m.Viewport.TotalLineCount())

		jump := m.Viewport.AtBottom()
		if jump {
			lines := strings.Split(m.SessionHandler.ActiveSession().Content, "\n")
			if len(lines) > 1000 {
				m.SessionHandler.ActiveSession().Content = strings.Join(lines[len(lines)-1000:], "\n")
			}
		}
		m.Viewport.SetContent(m.SessionHandler.ActiveSession().Content)
		if jump {
			m.Viewport.GotoBottom()
		}

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

		// update map?
		if m.SessionHandler.ActiveSession().Connected && m.RightSideBarActive {
			if k, ok := m.SessionHandler.Plugins.Plugins["kallisti"]; ok {
				tp, err := k.Plugin.Lookup("MakeMap")
				if err == nil {
					m.RightSideBar.SetContent(tp.(func(*session.Session, int, int) string)(m.SessionHandler.ActiveSession(), RightSideBarSize, 20))
				} else {
					m.RightSideBar.SetContent("Lookup failure")
				}
			}
		}

		cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))

	case session.TextinputMsg:
		if msg.Toggle_password {
			if msg.Password_mode {
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
			m.SessionHandler.ActiveSession().HandleInput(order)
			m.Input.SetValue("")
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
				width -= LeftSideBarSize
			}
			if m.RightSideBarActive {
				width -= RightSideBarSize
			}

			m.Viewport = viewport.New(width, msg.Height-verticalMarginHeight)
			m.Viewport.YPosition = 0
			m.Viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.Viewport.SetContent(session.Motd())
			m.Ready = true

			m.LeftSideBar = viewport.New(LeftSideBarSize, msg.Height-verticalMarginHeight)
			m.LeftSideBar.YPosition = 0
			m.LeftSideBar.HighPerformanceRendering = useHighPerformanceRenderer
			m.LeftSideBar.SetContent("LEFT")

			m.RightSideBar = viewport.New(RightSideBarSize, msg.Height-verticalMarginHeight)
			m.RightSideBar.YPosition = 0
			m.RightSideBar.HighPerformanceRendering = useHighPerformanceRenderer
			m.RightSideBar.SetContent("RIGHT")

			m.Input.Cursor.BlinkSpeed = 500 * time.Millisecond
			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			//m.viewport.YPosition = headerHeight + 1
		} else {
			width := msg.Width
			if m.LeftSideBarActive {
				width -= LeftSideBarSize
			}
			if m.RightSideBarActive {
				width -= RightSideBarSize
			}

			m.Viewport.Width = width

			m.Viewport.Height = msg.Height - verticalMarginHeight
			m.LeftSideBar.Width = LeftSideBarSize
			m.LeftSideBar.SetContent("LEFT")
			m.LeftSideBar.Height = msg.Height - verticalMarginHeight

			m.RightSideBar.Width = RightSideBarSize
			if k, ok := m.SessionHandler.Plugins.Plugins["kallisti"]; ok {
				tp, err := k.Plugin.Lookup("MakeMap")
				if err == nil {
					m.RightSideBar.SetContent(tp.(func(*session.Session, int, int) string)(m.SessionHandler.ActiveSession(), RightSideBarSize, 20))
				} else {
					m.RightSideBar.SetContent("Lookup failure")
				}
			}

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

		//default:
		//	log.Printf("Unknown message type: %s", reflect.TypeOf(msg))
	}

	return m, tea.Batch(cmds...)
}

func main() {
	var kallistiFlag = flag.Bool("kallisti", false, "Use Kallisti plugin")
	var helpFlag = flag.Bool("help", false, "Show help")

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}

	defer f.Close()

	m := ZifModel{Input: textinput.New(), SessionHandler: session.NewHandler()}
	m.SessionHandler.Active = "zif"
	m.Input.Placeholder = "Welcome to Zif, type #HELP to get started"
	m.Input.Focus()
	m.Input.CharLimit = 156
	m.Input.Width = 20

	if *kallistiFlag {

		p, err := plugin.Open("./kallisti.so")
		if err != nil {
			fmt.Printf("Error locating kallisti plugin: %s.\n", err.Error())
			os.Exit(1)
		}
		v, err := p.Lookup("Info")
		var version string
		if err != nil {
			version = v.(session.PluginInfo).Version
		} else {
			version = "unknown"
		}
		m.SessionHandler.Plugins.Plugins["kallisti"] = session.PluginInfo{Plugin: p, Name: "Kallisti", Version: version, Description: "Legends of Kallisti convenience add-ons"}

	}

	if len(m.SessionHandler.Plugins.Plugins) > 0 {
		m.SessionHandler.ActiveSession().Output("Installed Plugins:\n" + m.SessionHandler.PluginMOTD() + "\n")
	}

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
