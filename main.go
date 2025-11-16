package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/perlsaiyan/zif/config"
	"github.com/perlsaiyan/zif/layout"
	"github.com/perlsaiyan/zif/session"

	"github.com/mistakenelf/teacup/statusbar"
)

const useHighPerformanceRenderer = false

type ZifModel struct {
	Name           string
	Config         *config.Config
	Plugins        []*plugin.Plugin
	Input          textinput.Model
	Viewport       viewport.Model // Kept for backward compatibility during transition
	Layout         *layout.Layout // Flexible layout system
	SessionHandler session.SessionHandler
	StatusBar      statusbar.Model
	Ready          bool
	Error          string // Track panic/error messages
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

	var content string
	// Always use new layout system
	if m.Layout != nil {
		content = m.Layout.Render()
	} else {
		// Fallback to single viewport if layout not initialized
		content = m.Viewport.View()
	}

	// Display error/panic message if present
	if m.Error != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			Background(lipgloss.Color("#000000")).
			Padding(0, 1).
			Bold(true)
		errorBox := errorStyle.Render("ERROR: " + m.Error)
		content = lipgloss.JoinVertical(lipgloss.Top, errorBox, content)
	}

	return lipgloss.JoinVertical(lipgloss.Top, content, m.Input.View(), m.StatusBar.View())
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

// logPanic writes panic information to a panic log file
func logPanic(location string, panicValue interface{}, stack []byte) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "zif")
		os.MkdirAll(configDir, 0755)
	}
	
	panicLogPath := filepath.Join(configDir, "panic.log")
	
	panicInfo := fmt.Sprintf("=== PANIC at %s ===\nTime: %s\nPanic: %v\n\nStack trace:\n%s\n\n",
		location, time.Now().Format(time.RFC3339), panicValue, string(stack))
	
	// Append to panic log file
	if f, err := os.OpenFile(panicLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, panicInfo)
		f.Close()
	}
	
	// Also log to standard log
	log.Printf("PANIC in %s: %v\nStack:\n%s", location, panicValue, string(stack))
}

func (m ZifModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		//cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Recover from panics in Update
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			logPanic("Update", r, stack)
			
			errMsg := fmt.Sprintf("PANIC in Update: %v\n(Check ~/.config/zif/panic.log for details)", r)
			// Try to output to active session if available
			if m.SessionHandler.ActiveSession() != nil {
				m.SessionHandler.ActiveSession().Output("\n" + errMsg)
			}
			// Store error in model for display
			m.Error = errMsg
		}
	}()

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

		// Update layout system
		if m.Layout != nil {
			mainPane := m.Layout.FindPane("main")
			if mainPane != nil {
				mainPane.Viewport.SetContent(m.SessionHandler.ActiveSession().Content)
				mainPane.Viewport.GotoBottom()
				m.StatusBar.ThirdColumn = fmt.Sprintf("%d", mainPane.Viewport.TotalLineCount())
			}
			// Ensure map pane exists if kallisti is active
			m.ensureMapPane()
		}

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

		// Update content in layout system
		if m.Layout != nil {
			// Find the main viewport pane or active pane
			mainPane := m.Layout.FindPane("main")
			if mainPane == nil {
				mainPane = m.Layout.GetActivePane()
			}
			if mainPane != nil {
				jump := mainPane.Viewport.AtBottom()
				if jump {
					lines := strings.Split(m.SessionHandler.ActiveSession().Content, "\n")
					if len(lines) > 1000 {
						m.SessionHandler.ActiveSession().Content = strings.Join(lines[len(lines)-1000:], "\n")
					}
				}
				mainPane.Viewport.SetContent(m.SessionHandler.ActiveSession().Content)
				if jump {
					mainPane.Viewport.GotoBottom()
				}
				m.StatusBar.ThirdColumn = fmt.Sprintf("%d", mainPane.Viewport.TotalLineCount())
			}

			// Update map pane if kallisti plugin is active
			m.updateMapPane()

			if useHighPerformanceRenderer {
				// Sync all panes' viewports
				for _, pane := range m.Layout.GetAllPanes() {
					if pane.Viewport.Width > 0 && pane.Viewport.Height > 0 {
						cmds = append(cmds, viewport.Sync(pane.Viewport))
					}
				}
			}
		}

		cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))

	case session.TextinputMsg:
		if msg.Toggle_password {
			if msg.Password_mode {
				log.Printf("Turning on password mode\n")
				m.Input.EchoMode = textinput.EchoPassword
				m.SessionHandler.ActiveSession().PasswordMode = true
			} else {
				log.Printf("Turning off password mode\n")
				m.Input.EchoMode = textinput.EchoNormal
				m.SessionHandler.ActiveSession().PasswordMode = false
			}

			cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))
		}

	case layout.LayoutCommandMsg:
		// Handle layout commands
		m.handleLayoutCommand(msg)
		cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))

	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" {
			return m, tea.Quit
		} else if k := msg.String(); k == "pgup" || k == "pgdown" || k == "end" || k == "home" {
			// Handle viewport scrolling in layout system
			if m.Layout != nil {
				activePane := m.Layout.GetActivePane()
				if activePane != nil {
					var viewcmd tea.Cmd
					switch k {
					case "end":
						activePane.Viewport.GotoBottom()
					case "home":
						activePane.Viewport.GotoTop()
					default:
						activePane.Viewport, viewcmd = activePane.Viewport.Update(msg)
					}
					cmds = append(cmds, viewcmd)
				}
			}
		} else if k := msg.String(); k == "enter" {
			m.Input.Placeholder = ""
			order := strings.TrimSpace(m.Input.Value())
			
			// Check for layout commands
			if strings.HasPrefix(order, "#") {
				cmdParts := strings.Fields(order[1:])
				if len(cmdParts) > 0 {
					cmdName := strings.ToLower(cmdParts[0])
					if cmdName == "split" || cmdName == "unsplit" || cmdName == "panes" || cmdName == "pane" || cmdName == "focus" {
						// Parse and handle layout command
						m.handleLayoutCommandFromString(order)
					}
				}
			}
			
			m.SessionHandler.ActiveSession().HandleInput(order)
			m.Input.SetValue("")
		} else {
			var inputcmd tea.Cmd
			m.Input, inputcmd = m.Input.Update(msg)
			cmds = append(cmds, inputcmd)
		}

	case tea.WindowSizeMsg:
		footerHeight := 2
		verticalMarginHeight := footerHeight

		// Initialize or update layout system
		if m.Layout == nil {
			m.Layout = layout.NewLayout("main")
			mainPane := m.Layout.FindPane("main")
			if mainPane != nil {
				mainPane.Viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
				mainPane.Viewport.HighPerformanceRendering = useHighPerformanceRenderer
				mainPane.Viewport.SetContent(session.Motd())
			}
			m.Ready = true

			// Auto-create map pane if kallisti plugin is active
			m.ensureMapPane()
		}

		m.Layout.SetSize(msg.Width, msg.Height-verticalMarginHeight)
		// Update all panes' viewport sizes
		for _, pane := range m.Layout.GetAllPanes() {
			if pane.Viewport.Width > 0 && pane.Viewport.Height > 0 {
				pane.Viewport.Width = pane.Width
				pane.Viewport.Height = pane.Height
			}
		}

		m.Input.Cursor.BlinkSpeed = 500 * time.Millisecond

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
			// Sync all panes' viewports
			for _, pane := range m.Layout.GetAllPanes() {
				if pane.Viewport.Width > 0 && pane.Viewport.Height > 0 {
					cmds = append(cmds, viewport.Sync(pane.Viewport))
				}
			}
		}

		//default:
		//	log.Printf("Unknown message type: %s", reflect.TypeOf(msg))
	}

	return m, tea.Batch(cmds...)
}

// handleLayoutCommand processes layout command messages
func (m *ZifModel) handleLayoutCommand(msg layout.LayoutCommandMsg) {
	if m.Layout == nil {
		m.Layout = layout.NewLayout("main")
	}

	// Cast session from interface{} to *session.Session
	s, ok := msg.Session.(*session.Session)
	if !ok {
		log.Printf("Invalid session type in layout command")
		return
	}

	switch msg.Command {
	case "split":
		if len(msg.Args) < 5 {
			s.Output("Invalid split command\n")
			return
		}
		paneID := msg.Args[0]
		newPaneID := msg.Args[1]
		directionStr := msg.Args[2]
		splitPercentStr := msg.Args[3]
		paneTypeStr := msg.Args[4]

		var direction layout.SplitDirection
		if directionStr == "horizontal" {
			direction = layout.SplitHorizontal
		} else {
			direction = layout.SplitVertical
		}

		splitPercent, err := strconv.Atoi(splitPercentStr)
		if err != nil {
			s.Output(fmt.Sprintf("Invalid split percentage: %s\n", splitPercentStr))
			return
		}

		paneType := layout.ParsePaneType(paneTypeStr)

		err = m.Layout.Split(paneID, direction, splitPercent, newPaneID, paneType)
		if err != nil {
			s.Output(fmt.Sprintf("Error splitting pane: %v\n", err))
			return
		}

		// Initialize the new pane's viewport
		newPane := m.Layout.FindPane(newPaneID)
		if newPane != nil {
			newPane.Viewport = viewport.New(0, 0)
			newPane.Viewport.HighPerformanceRendering = useHighPerformanceRenderer
			newPane.Title = string(paneType)
		}

		s.Output(fmt.Sprintf("Split pane %s, created %s\n", paneID, newPaneID))

	case "unsplit":
		if len(msg.Args) < 1 {
			s.Output("Invalid unsplit command\n")
			return
		}
		paneID := msg.Args[0]
		err := m.Layout.Unsplit(paneID)
		if err != nil {
			s.Output(fmt.Sprintf("Error unsplitting pane: %v\n", err))
			return
		}
		s.Output(fmt.Sprintf("Removed pane %s\n", paneID))

	case "list":
		list := m.Layout.ListPanes()
		s.Output(list + "\n")

	case "info":
		if len(msg.Args) < 1 {
			s.Output("Usage: #pane [pane_id]\n")
			return
		}
		paneID := msg.Args[0]
		info := m.Layout.GetPaneInfo(paneID)
		s.Output(info + "\n")

	case "focus":
		if len(msg.Args) < 1 {
			s.Output("Usage: #focus [pane_id]\n")
			return
		}
		paneID := msg.Args[0]
		err := m.Layout.SetActivePane(paneID)
		if err != nil {
			s.Output(fmt.Sprintf("Error focusing pane: %v\n", err))
			return
		}
		s.Output(fmt.Sprintf("Focused pane %s\n", paneID))
	}
}

// handleLayoutCommandFromString parses a command string and handles layout commands
func (m *ZifModel) handleLayoutCommandFromString(cmd string) {
	// Layout commands are now handled directly in session/commands.go
	// This function is kept for potential future use
}

// ensureMapPane creates a map pane if kallisti plugin is active and map pane doesn't exist
func (m *ZifModel) ensureMapPane() {
	if m.Layout == nil {
		return
	}

	// Check if map pane already exists
	mapPane := m.Layout.FindPane("map")
	if mapPane != nil {
		return
	}

	// Check if kallisti plugin is active
	if _, ok := m.SessionHandler.Plugins.Plugins["kallisti"]; !ok {
		return
	}

	// Split main pane horizontally to add map pane on the right (30% for map, 70% for main)
	err := m.Layout.Split("main", layout.SplitHorizontal, 70, "map", layout.PaneTypeSidebar)
	if err != nil {
		log.Printf("Error creating map pane: %v", err)
		return
	}

	// Get the map pane and set it up
	mapPane = m.Layout.FindPane("map")
	if mapPane != nil {
		mapPane.Viewport = viewport.New(0, 0)
		mapPane.Viewport.HighPerformanceRendering = useHighPerformanceRenderer
		mapPane.Title = "Map"
		mapPane.MinWidth = 30
		// Set custom render function to display map content
		mapPane.RenderFunc = func(p *layout.Pane, width, height int) string {
			if p.Content != "" {
				// Use viewport for scrolling if content is large
				if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
					p.Viewport.SetContent(p.Content)
					content := p.Viewport.View()
					if p.Title != "" {
						titleBar := lipgloss.NewStyle().
							Width(width).
							Border(lipgloss.NormalBorder()).
							BorderBottom(false).
							Render(p.Title)
						return lipgloss.JoinVertical(lipgloss.Top, titleBar, content)
					}
					return content
				}
				// Simple content display
				return p.Content
			}
			return "Map (loading...)"
		}
	}
}

// updateMapPane updates the map pane content if kallisti plugin is active
func (m *ZifModel) updateMapPane() {
	if m.Layout == nil {
		return
	}

	mapPane := m.Layout.FindPane("map")
	if mapPane == nil {
		// Try to create it if it doesn't exist
		m.ensureMapPane()
		mapPane = m.Layout.FindPane("map")
		if mapPane == nil {
			return
		}
	}

	// Ensure viewport is properly sized
	if mapPane.Viewport.Width != mapPane.Width || mapPane.Viewport.Height != mapPane.Height {
		mapPane.Viewport.Width = mapPane.Width
		mapPane.Viewport.Height = mapPane.Height
	}

	// Check if kallisti plugin is active and session is connected
	if k, ok := m.SessionHandler.Plugins.Plugins["kallisti"]; ok {
		if m.SessionHandler.ActiveSession().Connected {
			tp, err := k.Plugin.Lookup("MakeMap")
			if err == nil {
				// Calculate map size based on pane dimensions
				mapWidth := mapPane.Width
				if mapWidth < 10 {
					mapWidth = 50 // Default size
				}
				mapHeight := mapPane.Height
				if mapHeight < 5 {
					mapHeight = 20 // Default size
				}

				mapContent := tp.(func(*session.Session, int, int) string)(
					m.SessionHandler.ActiveSession(), mapWidth, mapHeight)
				mapPane.Content = mapContent
			}
		}
	}
}

func main() {
	var kallistiFlag = flag.Bool("kallisti", false, "Use Kallisti plugin")
	var helpFlag = flag.Bool("help", false, "Show help")
	var noAutostartFlag = flag.Bool("no-autostart", false, "Skip auto-loading sessions from sessions.yaml")

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

	// Load and auto-start sessions from sessions.yaml (unless --no-autostart flag is set)
	if !*noAutostartFlag {
		sessionsConfig, err := config.LoadSessionsConfig()
		if err != nil {
			log.Printf("Warning: failed to load sessions config: %v", err)
		} else {
			// Auto-start all configured sessions with autostart enabled
			for _, sessionConfig := range sessionsConfig.Sessions {
				if sessionConfig.Autostart && sessionConfig.Name != "" && sessionConfig.Address != "" {
					m.SessionHandler.AddSession(sessionConfig.Name, sessionConfig.Address)
				}
			}
			// Set the first session as active if any sessions were loaded
			if len(sessionsConfig.Sessions) > 0 && len(m.SessionHandler.Sessions) > 1 {
				// Find the first successfully created session with autostart enabled (not the default "zif" session)
				for _, sessionConfig := range sessionsConfig.Sessions {
					if sessionConfig.Autostart {
						if _, exists := m.SessionHandler.Sessions[sessionConfig.Name]; exists {
							m.SessionHandler.Active = sessionConfig.Name
							m.SessionHandler.Sub <- session.SessionChangeMsg{ActiveSession: m.SessionHandler.ActiveSession()}
							break
						}
					}
				}
			}
		}
	}

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

	// Recover from panics in the main program
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			logPanic("main", r, stack)
			
			errMsg := fmt.Sprintf("PANIC in main program: %v\n(Check ~/.config/zif/panic.log for details)", r)
			// Try to output to active session if available
			if m.SessionHandler.ActiveSession() != nil {
				m.SessionHandler.ActiveSession().Output("\n" + errMsg)
			}
			// Print to stderr as fallback
			fmt.Fprintf(os.Stderr, "\n%s\n", errMsg)
			fmt.Fprintf(os.Stderr, "Stack trace saved to ~/.config/zif/panic.log\n")
			fmt.Fprintf(os.Stderr, "Press Ctrl+C to exit...\n")
			// Wait a bit so user can see the error, then exit
			time.Sleep(5 * time.Second)
			os.Exit(1)
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
