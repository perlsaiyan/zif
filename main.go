package main

// An example program demonstrating the pager component from the Bubbles
// component library.

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"syscall"
	"unsafe"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	kallisti "github.com/perlsaiyan/zif/protocol"
	config "github.com/perlsaiyan/zif/config"
)

// You generally won't need this unless you're processing stuff with
// complicated ANSI escape sequences. Turn it on if you notice flickering.
//
// Also keep in mind that high performance rendering only works for programs
// that use the full size of the terminal. We're enabling that below with
// tea.EnterAltScreen().
const useHighPerformanceRenderer = true

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

type model struct {
	sub      chan tea.Msg
	content  string
	ready    bool
	viewport viewport.Model
	socket   net.Conn
	input    textinput.Model
	msdp     *kallisti.MSDPHandler
	config   *config.Config
}

type updateMessage struct {
	content string
}

type textinputMsg struct {
	password_mode   bool
	toggle_password bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		waitForActivity(m.sub), // wait for activity
		tea.SetWindowTitle("Flowtest"),
	)
}

// Handle triggers
func triggers(m *model, line string) {
	r, _ := regexp.Compile("^Enter your account name.")
	if len(m.config.Session.Username) > 0 && r.MatchString(line) {
		m.socket.Write([]byte(m.config.Session.Username + "\n"))
	}

	r, _ = regexp.Compile("^Please enter your account password")
	if len(m.config.Session.Password) > 0 && r.MatchString(line) {
		m.socket.Write([]byte(m.config.Session.Password + "\n"))
	}
}

// this only works on linux
// we'll need something special for windows and mac
func terminalEcho(show bool) {
	// Enable or disable echoing terminal input. This is useful specifically for
	// when users enter passwords.
	// calling terminalEcho(true) turns on echoing (normal mode)
	// calling terminalEcho(false) hides terminal input.
	var termios = &syscall.Termios{}
	var fd = os.Stdout.Fd()

	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd,
		syscall.TCGETS, uintptr(unsafe.Pointer(termios))); err != 0 {
		return
	}

	if show {
		termios.Lflag |= syscall.ECHO
	} else {
		termios.Lflag &^= syscall.ECHO
	}

	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd,
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(termios))); err != 0 {
		return
	}
}

// Read from the MUD stream, parse MSDP, etc
func mudReader(sub chan tea.Msg, socket net.Conn, m *model) tea.Cmd {

	buffer := make([]byte, 1)
	var outbuf string

	for {
		_, err := socket.Read(buffer)
		if err != nil {
			fmt.Println("Error: ", err)
			sub <- tea.KeyMsg.String
		}

		if buffer[0] == 255 {

			_, _ = socket.Read(buffer) // read one char for now to eat GA
			if buffer[0] == 249 {      //this is GO AHEAD
				log.Println("Got GA")
				sub <- updateMessage{content: outbuf + "\n"}
				triggers(m, outbuf)
				outbuf = ""
			} else if buffer[0] == 251 { // WILL
				_, _ = socket.Read(buffer)
				log.Println("Debug WILL:", buffer)
				if buffer[0] == 1 { // ECHO / password mask
					log.Printf("Got password mask request")
					//sub <- textinputMsg{password_mode: true, toggle_password: true}
				} else if buffer[0] == 69 {
					log.Printf("Offered MSDP, accepting")
					buf := []byte{255, 253, 69, 255, kallisti.SB, kallisti.MSDP, kallisti.MSDP_VAR, 'L', 'I', 'S', 'T',
						kallisti.MSDP_VAL, 'C', 'O', 'M', 'M', 'A', 'N', 'D', 'S', 255, kallisti.SE}
					m.socket.Write(buf)
					m.msdp.HandleWill(m.socket)

				} else {
					log.Printf("SERVER WILL %v\n", buffer)
				}
			} else if buffer[0] == 252 { // WONT
				_, _ = socket.Read(buffer)
				if buffer[0] == 1 {
					log.Printf("Got password unmask request")
					//sub <- textinputMsg{password_mode: false, toggle_password: true}
				} else {
					log.Printf("SERVER WONT %v\n", buffer)
				}
			} else if buffer[0] == 253 { // DO
				_, _ = socket.Read(buffer)
				log.Printf("Got DO %v", buffer)
				if buffer[0] == 24 { // TERM TYPE
					buf := []byte{255, 251, 24}
					log.Printf("Sending %v", buf)
					socket.Write(buf)
				}
			} else if buffer[0] == 254 { // DONT
				_, _ = socket.Read(buffer)
				log.Printf("Got DONT %v", buffer)
			} else if buffer[0] == kallisti.SB {

				var sb []byte
				for {
					_, _ = socket.Read(buffer)
					if buffer[0] == kallisti.SE {
						break
					}
					sb = append(sb, buffer...)
				}
				log.Printf("Good SB: %v", sb)
				switch sb[0] {
				case 69:
					m.msdp.HandleSB(socket, sb)
				}
			} else {
				log.Printf("Unknown IAC %v\n", buffer[0])
			}
		} else if buffer[0] == 10 {
			// newline, print big buf and go
			triggers(m, outbuf)
			sub <- updateMessage{content: outbuf + "\n"}
			outbuf = ""
		} else {
			outbuf += string(buffer[0])
		}

	}

}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return tea.Msg(<-sub)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case textinputMsg:
		if msg.toggle_password {
			if msg.password_mode {
				log.Printf("Turning on password mode\n")
				m.input.EchoMode = textinput.EchoPassword

			} else {
				log.Printf("Turning off password mode\n")
				m.input.EchoMode = textinput.EchoNormal
			}

			var icmd tea.Cmd
			m.input, icmd = m.input.Update(msg)
			cmds = append(cmds, waitForActivity(m.sub))
			cmds = append(cmds, icmd)
			return m, tea.Sequence(cmds...)
		}

	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" {
			return m, tea.Quit
		} else if k := msg.String(); k == "pgup" || k == "pgdown" || k == "end" || k == "home" {
			var viewcmd tea.Cmd
			m.viewport, viewcmd = m.viewport.Update(msg)
			cmds = append(cmds, viewcmd)
		} else if k := msg.String(); k == "enter" {
			order := strings.TrimSpace(m.input.Value())
			if len(order) > 0 {
				if order[0] == '#' {
					if order == "#MSDP" {
						msdp_vals := fmt.Sprintf("MSDP: %+v", m.msdp)
						m.sub <- updateMessage{content: msdp_vals}
					}

				} else {
					m.socket.Write([]byte(m.input.Value() + "\n"))
				}
			} else {
				m.socket.Write([]byte("\n"))
			}
			m.input.Reset()
		} else {
			var inputcmd tea.Cmd
			m.input, inputcmd = m.input.Update(msg)
			cmds = append(cmds, inputcmd)
		}

	case updateMessage:
		m.content += msg.content
		jump := false
		if m.viewport.AtBottom() {
			jump = true
		}

		m.viewport.SetContent(m.content)

		if jump {
			m.viewport.GotoBottom()
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
		cmds = append(cmds, waitForActivity(m.sub))

	case tea.WindowSizeMsg:
		//headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		//verticalMarginHeight := headerHeight + footerHeight
		verticalMarginHeight := footerHeight

		if !m.ready {
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

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	}

	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	//return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	return fmt.Sprintf("%s\n%s", m.viewport.View(), m.footerView())
}

func (m model) headerView() string {
	title := titleStyle.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	return m.input.View()
	//return lipgloss.JoinHorizontal(lipgloss.Center, line)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {

	c := config.GetConfig()

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	// Load some text for our viewport
	content := ""
	if len(c.Session.Hostname) == 0 {
		fmt.Printf("Please set Hostname in config file.\n")
		os.Exit(1)
	}
	conn, err := net.Dial("tcp", c.Session.Hostname)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	m := model{content: string(content), sub: make(chan tea.Msg), socket: conn, input: textinput.New(), msdp: kallisti.NewMSDP()}
	m.config = c
	m.input.Placeholder = "Welcome to Kallisti"
	m.input.Focus()
	m.input.CharLimit = 156
	//m.input.Width = 20

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	go mudReader(m.sub, m.socket, &m)

	if _, err := p.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
