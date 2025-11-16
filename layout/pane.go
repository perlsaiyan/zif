package layout

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaneType represents the type of content a pane displays
type PaneType string

const (
	PaneTypeViewport PaneType = "viewport" // Main MUD output
	PaneTypeComms    PaneType = "comms"    // Communications/logs
	PaneTypeSidebar  PaneType = "sidebar"  // Information sidebar
	PaneTypeGraph    PaneType = "graph"    // Graph/chart display
	PaneTypeCustom   PaneType = "custom"   // Custom content
)

// SplitDirection indicates how a container is split
type SplitDirection string

const (
	SplitHorizontal SplitDirection = "horizontal" // Split left/right
	SplitVertical   SplitDirection = "vertical"   // Split top/bottom
)

// Pane represents a single display pane in the layout
type Pane struct {
	ID          string
	Type        PaneType
	Viewport    viewport.Model
	Content     string
	MinWidth    int
	MinHeight   int
	Width       int // Current width (for leaf panes)
	Height      int // Current height (for leaf panes)
	Style       lipgloss.Style
	Title       string
	Visible     bool
	RenderFunc  func(*Pane, int, int) string // Custom render function
}

// Container represents a split container with child panes
type Container struct {
	Direction SplitDirection
	Split     int // Split position (0-100, percentage)
	Left      Node
	Right     Node
	MinWidth  int
	MinHeight int
	Width     int
	Height    int
}

// Node is either a Pane or a Container
type Node interface {
	Render(width, height int) string
	GetMinWidth() int
	GetMinHeight() int
	SetSize(width, height int)
	FindPane(id string) *Pane
	GetAllPanes() []*Pane
	Update(msg tea.Msg) (Node, tea.Cmd)
}

// Pane methods
func (p *Pane) Render(width, height int) string {
	if !p.Visible {
		return ""
	}

	p.Width = width
	p.Height = height

	// Update viewport size
	if p.Viewport.Width != width || p.Viewport.Height != height {
		p.Viewport.Width = width
		p.Viewport.Height = height
	}

	// Use custom render function if available
	if p.RenderFunc != nil {
		return p.RenderFunc(p, width, height)
	}

	// Default rendering
	if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
		content := p.Viewport.View()
		if p.Title != "" {
			titleBar := p.Style.Copy().
				Width(width).
				Border(lipgloss.NormalBorder()).
				BorderBottom(false).
				Render(p.Title)
			content = lipgloss.JoinVertical(lipgloss.Top, titleBar, content)
		}
		return content
	}

	return p.Style.Copy().Width(width).Height(height).Render(p.Content)
}

func (p *Pane) GetMinWidth() int {
	if p.MinWidth > 0 {
		return p.MinWidth
	}
	return 10
}

func (p *Pane) GetMinHeight() int {
	if p.MinHeight > 0 {
		return p.MinHeight
	}
	return 5
}

func (p *Pane) SetSize(width, height int) {
	p.Width = width
	p.Height = height
	if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
		p.Viewport.Width = width
		p.Viewport.Height = height
	}
}

func (p *Pane) FindPane(id string) *Pane {
	if p.ID == id {
		return p
	}
	return nil
}

func (p *Pane) GetAllPanes() []*Pane {
	return []*Pane{p}
}

func (p *Pane) Update(msg tea.Msg) (Node, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Window size is handled at the layout level
	case tea.KeyMsg:
		// Pass key messages to viewport for scrolling
		if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
			p.Viewport, cmd = p.Viewport.Update(msg)
		}
	}
	return p, cmd
}

// Container methods
func (c *Container) Render(width, height int) string {
	c.Width = width
	c.Height = height

	// Calculate split position
	splitPos := c.Split
	if splitPos < 0 {
		splitPos = 0
	}
	if splitPos > 100 {
		splitPos = 100
	}

	var leftWidth, leftHeight, rightWidth, rightHeight int

	if c.Direction == SplitHorizontal {
		// Split left/right
		leftWidth = (width * splitPos) / 100
		rightWidth = width - leftWidth
		leftHeight = height
		rightHeight = height

		// Ensure minimum sizes
		leftMin := c.Left.GetMinWidth()
		rightMin := c.Right.GetMinWidth()

		if leftWidth < leftMin {
			leftWidth = leftMin
			rightWidth = width - leftWidth
		}
		if rightWidth < rightMin {
			rightWidth = rightMin
			leftWidth = width - rightWidth
		}

		left := c.Left.Render(leftWidth, leftHeight)
		right := c.Right.Render(rightWidth, rightHeight)

		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	} else {
		// Split top/bottom
		leftHeight = (height * splitPos) / 100
		rightHeight = height - leftHeight
		leftWidth = width
		rightWidth = width

		// Ensure minimum sizes
		leftMin := c.Left.GetMinHeight()
		rightMin := c.Right.GetMinHeight()

		if leftHeight < leftMin {
			leftHeight = leftMin
			rightHeight = height - leftHeight
		}
		if rightHeight < rightMin {
			rightHeight = rightMin
			leftHeight = height - rightHeight
		}

		top := c.Left.Render(leftWidth, leftHeight)
		bottom := c.Right.Render(rightWidth, rightHeight)

		return lipgloss.JoinVertical(lipgloss.Top, top, bottom)
	}
}

func (c *Container) GetMinWidth() int {
	if c.Direction == SplitHorizontal {
		return c.Left.GetMinWidth() + c.Right.GetMinWidth()
	}
	min := c.Left.GetMinWidth()
	if c.Right.GetMinWidth() > min {
		min = c.Right.GetMinWidth()
	}
	return min
}

func (c *Container) GetMinHeight() int {
	if c.Direction == SplitVertical {
		return c.Left.GetMinHeight() + c.Right.GetMinHeight()
	}
	min := c.Left.GetMinHeight()
	if c.Right.GetMinHeight() > min {
		min = c.Right.GetMinHeight()
	}
	return min
}

func (c *Container) SetSize(width, height int) {
	c.Width = width
	c.Height = height
	// Children will be sized during Render
}

func (c *Container) FindPane(id string) *Pane {
	if p := c.Left.FindPane(id); p != nil {
		return p
	}
	return c.Right.FindPane(id)
}

func (c *Container) GetAllPanes() []*Pane {
	var panes []*Pane
	panes = append(panes, c.Left.GetAllPanes()...)
	panes = append(panes, c.Right.GetAllPanes()...)
	return panes
}

func (c *Container) Update(msg tea.Msg) (Node, tea.Cmd) {
	var cmds []tea.Cmd
	var leftCmd, rightCmd tea.Cmd

	c.Left, leftCmd = c.Left.Update(msg)
	c.Right, rightCmd = c.Right.Update(msg)

	if leftCmd != nil {
		cmds = append(cmds, leftCmd)
	}
	if rightCmd != nil {
		cmds = append(cmds, rightCmd)
	}

	return c, tea.Batch(cmds...)
}

// NewPane creates a new pane
func NewPane(id string, paneType PaneType) *Pane {
	vp := viewport.New(0, 0)
	return &Pane{
		ID:       id,
		Type:     paneType,
		Viewport: vp,
		Content:  "",
		Visible:  true,
		Style:    lipgloss.NewStyle(),
		MinWidth: 10,
		MinHeight: 5,
	}
}

// NewContainer creates a new split container
func NewContainer(direction SplitDirection, split int, left, right Node) *Container {
	return &Container{
		Direction: direction,
		Split:     split,
		Left:      left,
		Right:     right,
	}
}

// SplitPane splits a pane into a container with two panes
func SplitPane(pane *Pane, direction SplitDirection, split int, newPaneID string, newPaneType PaneType) *Container {
	newPane := NewPane(newPaneID, newPaneType)
	return NewContainer(direction, split, pane, newPane)
}

// Helper function to generate unique pane IDs
var paneIDCounter = 0

func GeneratePaneID(prefix string) string {
	paneIDCounter++
	return fmt.Sprintf("%s_%d", prefix, paneIDCounter)
}

// ParsePaneType converts a string to PaneType
func ParsePaneType(s string) PaneType {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "viewport", "main", "mud":
		return PaneTypeViewport
	case "comms", "comm", "log", "logs":
		return PaneTypeComms
	case "sidebar", "side", "info":
		return PaneTypeSidebar
	case "graph", "chart":
		return PaneTypeGraph
	default:
		return PaneTypeCustom
	}
}

