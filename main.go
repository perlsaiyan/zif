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

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
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
	LastMapVnum    string // Track last VNUM to avoid unnecessary map redraws
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
	} else if m.Viewport.Width > 0 && m.Viewport.Height > 0 {
		// Fallback to single viewport if layout not initialized and viewport is ready
		content = m.Viewport.View()
	} else {
		// Both layout and viewport are uninitialized
		content = "\n  Waiting for window size..."
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

// wrapViewportContent wraps content for a viewport based on its width.
// This ensures long lines wrap instead of running off the end.
func wrapViewportContent(content string, viewportWidth int) string {
	if viewportWidth <= 0 {
		// If width not set, return content as-is
		return content
	}

	// Split content into lines, wrap each line, then rejoin
	lines := strings.Split(content, "\n")
	wrappedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		if len(line) == 0 {
			// Preserve empty lines
			wrappedLines = append(wrappedLines, "")
			continue
		}
		// Wrap the line using wordwrap (which is ANSI-aware)
		wrapped := wordwrap.String(line, viewportWidth)
		wrappedLines = append(wrappedLines, wrapped)
	}

	return strings.Join(wrappedLines, "\n")
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
		// Check if ActiveSession is nil before accessing its fields
		if msg.ActiveSession == nil {
			log.Printf("Warning: SessionChangeMsg received with nil ActiveSession")
			// Try to get a valid active session
			activeSession := m.SessionHandler.ActiveSession()
			if activeSession == nil {
				// No valid session available, keep current state
				cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))
				return m, tea.Batch(cmds...)
			}
			// Use the valid session we found
			msg.ActiveSession = activeSession
		}

		log.Printf("Setting active session to %s (test: %s)", msg.ActiveSession.Name, m.SessionHandler.Active)
		m.SessionHandler.Active = msg.ActiveSession.Name

		m.StatusBar.FirstColumn = m.SessionHandler.Active
		activeSession := m.SessionHandler.ActiveSession()
		if activeSession != nil {
			if activeSession.Connected {
				m.StatusBar.SecondColumn = activeSession.Address
			} else {
				m.StatusBar.SecondColumn = "Not Connected"
			}

			// Update layout system
			if m.Layout != nil {
				mainPane := m.Layout.FindPane("main")
				if mainPane != nil {
					// Wrap content before setting it to viewport
					wrappedContent := wrapViewportContent(activeSession.Content, mainPane.Viewport.Width)
					mainPane.Viewport.SetContent(wrappedContent)
					mainPane.Viewport.GotoBottom()
					m.StatusBar.ThirdColumn = fmt.Sprintf("%d", mainPane.Viewport.TotalLineCount())
				}
				// Ensure map pane exists if kallisti is active
				m.ensureMapPane()
			}
		} else {
			// Fallback if active session is somehow nil
			m.StatusBar.SecondColumn = "No Session"
			log.Printf("Warning: ActiveSession() returned nil after setting active to %s", m.SessionHandler.Active)
		}

		cmds = append(cmds, waitForActivity(m.SessionHandler.Sub))

	case session.UpdateMessage:
		m.StatusBar.FirstColumn = m.SessionHandler.Active
		activeSession := m.SessionHandler.ActiveSession()
		if activeSession != nil {
			if activeSession.Connected {
				roomName := activeSession.MSDP.GetString("ROOM_NAME")
				if len(roomName) > 0 {
					if k, ok := m.SessionHandler.Plugins.Plugins["kallisti"]; ok {
						tp, err := k.Plugin.Lookup("TravelProgress")
						if err == nil {
							m.StatusBar.SecondColumn = tp.(func(*session.Session) string)(activeSession) +
								" " + roomName
						} else {
							m.StatusBar.SecondColumn = roomName
						}
					} else {
						m.StatusBar.SecondColumn = roomName
					}
				} else {
					m.StatusBar.SecondColumn = activeSession.Address
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
					contentToSet := activeSession.Content
					if jump {
						lines := strings.Split(contentToSet, "\n")
						if len(lines) > 1000 {
							contentToSet = strings.Join(lines[len(lines)-1000:], "\n")
						}
					}
					// Wrap content before setting it to viewport
					wrappedContent := wrapViewportContent(contentToSet, mainPane.Viewport.Width)
					mainPane.Viewport.SetContent(wrappedContent)
					if jump {
						mainPane.Viewport.GotoBottom()
					}
					m.StatusBar.ThirdColumn = fmt.Sprintf("%d", mainPane.Viewport.TotalLineCount())
				}

				// Update map pane if kallisti plugin is active
				m.updateMapPane()
			}
		} else {
			m.StatusBar.SecondColumn = "No Session"
		}

		// Continue with viewport sync even if session is nil
		if m.Layout != nil {

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
			activeSession := m.SessionHandler.ActiveSession()
			if activeSession != nil {
				if msg.Password_mode {
					log.Printf("Turning on password mode\n")
					m.Input.EchoMode = textinput.EchoPassword
					activeSession.PasswordMode = true
				} else {
					log.Printf("Turning off password mode\n")
					m.Input.EchoMode = textinput.EchoNormal
					activeSession.PasswordMode = false
				}
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

			activeSession := m.SessionHandler.ActiveSession()
			if activeSession != nil {
				activeSession.HandleInput(order)
			}
			m.Input.SetValue("")
		} else {
			var inputcmd tea.Cmd
			m.Input, inputcmd = m.Input.Update(msg)
			cmds = append(cmds, inputcmd)
		}

	case tea.MouseMsg:
		// Pass mouse events to layout for drag handling
		if m.Layout != nil {
			var layoutCmd tea.Cmd
			layoutCmd = m.Layout.Update(msg)
			if layoutCmd != nil {
				cmds = append(cmds, layoutCmd)
			}
		}
		// Also pass to input for potential mouse interactions
		var inputcmd tea.Cmd
		m.Input, inputcmd = m.Input.Update(msg)
		cmds = append(cmds, inputcmd)

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
				// Wrap MOTD content before setting it
				wrappedMotd := wrapViewportContent(session.Motd(), mainPane.Viewport.Width)
				mainPane.Viewport.SetContent(wrappedMotd)
			}
			m.Ready = true

			// Auto-create map pane if kallisti plugin is active
			m.ensureMapPane()
		}

		m.Layout.SetSize(msg.Width, msg.Height-verticalMarginHeight)
		// Update all panes' viewport sizes (accounting for actual borders)
		for _, pane := range m.Layout.GetAllPanes() {
			if pane.Viewport.Width > 0 && pane.Viewport.Height > 0 {
				widthReduction, heightReduction := pane.CalculateBorderReduction()
				viewportWidth := pane.Width - widthReduction
				viewportHeight := pane.Height - heightReduction
				if viewportWidth < 0 {
					viewportWidth = 0
				}
				if viewportHeight < 0 {
					viewportHeight = 0
				}
				pane.Viewport.Width = viewportWidth
				pane.Viewport.Height = viewportHeight
			}
		}

		m.Input.Cursor.BlinkSpeed = 500 * time.Millisecond

		m.StatusBar.Height = 1
		m.StatusBar.SetSize(msg.Width)
		activeSession := m.SessionHandler.ActiveSession()
		connected := func() string {
			if activeSession != nil && activeSession.Connected {
				return "✓"
			} else {
				return "✗"
			}
		}
		sessionName := "No Session"
		if activeSession != nil {
			sessionName = activeSession.Name
		}
		m.StatusBar.SetContent(sessionName, "Not Connected", "100% Efficient", connected())
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

	case "set_content":
		if len(msg.Args) < 2 {
			s.Output("Invalid set_content command\n")
			return
		}
		paneID := msg.Args[0]
		content := msg.Args[1]
		pane := m.Layout.FindPane(paneID)
		if pane == nil {
			s.Output(fmt.Sprintf("Pane %s not found\n", paneID))
			return
		}
		// Set content and update viewport if it exists
		pane.Content = content
		if pane.Viewport.Width > 0 && pane.Viewport.Height > 0 {
			// Wrap content before setting it to viewport
			wrappedContent := wrapViewportContent(content, pane.Viewport.Width)
			pane.Viewport.SetContent(wrappedContent)
			pane.Viewport.GotoTop()
		}
	case "set_border":
		if len(msg.Args) < 2 {
			s.Output("Invalid set_border command\n")
			return
		}
		paneID := msg.Args[0]
		borderType := msg.Args[1]
		color := ""
		if len(msg.Args) >= 3 {
			color = msg.Args[2]
		}
		pane := m.Layout.FindPane(paneID)
		if pane == nil {
			s.Output(fmt.Sprintf("Pane %s not found\n", paneID))
			return
		}
		pane.SetBorderStyle(borderType, color)
	case "progress_create":
		if len(msg.Args) < 1 {
			s.Output("Invalid progress_create command\n")
			return
		}
		paneID := msg.Args[0]
		width := 40 // Default width
		if len(msg.Args) >= 2 {
			var err error
			width, err = strconv.Atoi(msg.Args[1])
			if err != nil {
				s.Output(fmt.Sprintf("Invalid width: %s\n", msg.Args[1]))
				return
			}
		}
		pane := m.Layout.FindPane(paneID)
		if pane == nil {
			s.Output(fmt.Sprintf("Pane %s not found\n", paneID))
			return
		}
		// Create progress bar
		pane.ProgressBar = progress.New(progress.WithDefaultGradient())
		// Set width based on pane's actual width if available, otherwise use requested width
		if pane.Width > 0 {
			// Account for borders
			widthReduction, _ := pane.CalculateBorderReduction()
			progressWidth := pane.Width - widthReduction - 4 // 4 for padding
			if progressWidth < 10 {
				progressWidth = 10
			}
			if progressWidth > 80 {
				progressWidth = 80
			}
			pane.ProgressBar.Width = progressWidth
		} else {
			pane.ProgressBar.Width = width
		}
		pane.ProgressBar.ShowPercentage = true
		pane.ProgressPercent = 0.0
		pane.ShowProgress = true
		// Initialize with 0% to trigger first render
		pane.ProgressBar.SetPercent(0.0)

		// Debug output
		log.Printf("DEBUG: Created progress bar in pane %s: Width=%d, PaneWidth=%d, ShowProgress=%v, ProgressPercent=%f",
			paneID, pane.ProgressBar.Width, pane.Width, pane.ShowProgress, pane.ProgressPercent)

		s.Output(fmt.Sprintf("Created progress bar in pane %s (width: %d, pane width: %d)\n",
			paneID, pane.ProgressBar.Width, pane.Width))
	case "progress_update":
		if len(msg.Args) < 2 {
			s.Output("Invalid progress_update command\n")
			return
		}
		paneID := msg.Args[0]
		percentStr := msg.Args[1]
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			s.Output(fmt.Sprintf("Invalid percent: %s\n", percentStr))
			return
		}
		// Clamp percent to 0.0-1.0
		if percent < 0.0 {
			percent = 0.0
		}
		if percent > 1.0 {
			percent = 1.0
		}
		pane := m.Layout.FindPane(paneID)
		if pane == nil {
			s.Output(fmt.Sprintf("Pane %s not found\n", paneID))
			return
		}
		if !pane.ShowProgress {
			s.Output(fmt.Sprintf("Pane %s does not have a progress bar\n", paneID))
			return
		}
		// Update progress
		pane.ProgressPercent = percent
		// Set the progress bar value (animation will happen via frame messages in Update loop)
		pane.ProgressBar.SetPercent(percent)

		// Debug output
		log.Printf("DEBUG: Updated progress bar in pane %s: Percent=%f, Width=%d", paneID, percent, pane.ProgressBar.Width)
	case "progress_destroy":
		if len(msg.Args) < 1 {
			s.Output("Invalid progress_destroy command\n")
			return
		}
		paneID := msg.Args[0]
		pane := m.Layout.FindPane(paneID)
		if pane == nil {
			s.Output(fmt.Sprintf("Pane %s not found\n", paneID))
			return
		}
		if !pane.ShowProgress {
			s.Output(fmt.Sprintf("Pane %s does not have a progress bar\n", paneID))
			return
		}
		// Destroy progress bar
		pane.ShowProgress = false
		pane.ProgressPercent = 0.0
		// Reset progress bar model
		pane.ProgressBar = progress.New(progress.WithDefaultGradient())
		s.Output(fmt.Sprintf("Destroyed progress bar in pane %s\n", paneID))
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

	// Ensure viewport is properly sized (accounting for actual borders)
	widthReduction, heightReduction := mapPane.CalculateBorderReduction()
	viewportWidth := mapPane.Width - widthReduction
	viewportHeight := mapPane.Height - heightReduction
	if viewportWidth < 0 {
		viewportWidth = 0
	}
	if viewportHeight < 0 {
		viewportHeight = 0
	}
	if mapPane.Viewport.Width != viewportWidth || mapPane.Viewport.Height != viewportHeight {
		mapPane.Viewport.Width = viewportWidth
		mapPane.Viewport.Height = viewportHeight
	}

	// Check if kallisti plugin is active and session is connected
	activeSession := m.SessionHandler.ActiveSession()
	if activeSession != nil {
		if k, ok := m.SessionHandler.Plugins.Plugins["kallisti"]; ok {
			if activeSession.Connected {
				// Check if VNUM has changed
				currentVnum := activeSession.MSDP.GetString("ROOM_VNUM")
				if currentVnum == m.LastMapVnum && mapPane.Content != "" {
					// VNUM hasn't changed and we have content, skip redraw
					return
				}

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
						activeSession, mapWidth, mapHeight)
					mapPane.Content = mapContent
					m.LastMapVnum = currentVnum
				}
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
					err := m.SessionHandler.AddSession(sessionConfig.Name, sessionConfig.Address)
					if err != nil {
						log.Printf("Warning: failed to autostart session %s: %v", sessionConfig.Name, err)
					}
				}
			}
			// Set the first session as active if any sessions were loaded
			if len(sessionsConfig.Sessions) > 0 && len(m.SessionHandler.Sessions) > 1 {
				// Find the first successfully created session with autostart enabled (not the default "zif" session)
				for _, sessionConfig := range sessionsConfig.Sessions {
					if sessionConfig.Autostart {
						if _, exists := m.SessionHandler.Sessions[sessionConfig.Name]; exists {
							m.SessionHandler.Active = sessionConfig.Name
							activeSession := m.SessionHandler.ActiveSession()
							if activeSession != nil {
								m.SessionHandler.Sub <- session.SessionChangeMsg{ActiveSession: activeSession}
							}
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

	// Register plugins for all existing sessions (including default "zif" session)
	if len(m.SessionHandler.Plugins.Plugins) > 0 {
		for _, sess := range m.SessionHandler.Sessions {
			for _, v := range m.SessionHandler.Plugins.Plugins {
				log.Printf("Activating plugin %s for session %s", v.Name, sess.Name)
				f, err := v.Plugin.Lookup("RegisterSession")
				if err != nil {
					log.Printf("RegisterSession() lookup failure on plugin %s", v.Name)
					continue
				}
				f.(func(*session.Session))(sess)
			}
			// Inject context after plugins have registered their injectors
			if err := sess.InjectContext(); err != nil {
				log.Printf("Warning: failed to inject context for session %s: %v", sess.Name, err)
			}
		}

		activeSession := m.SessionHandler.ActiveSession()
		if activeSession != nil {
			activeSession.Output("Installed Plugins:\n" + m.SessionHandler.PluginMOTD() + "\n")
		}
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
