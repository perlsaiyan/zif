package layout

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
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
	// Border customization
	BorderStyle lipgloss.Border
	BorderColor lipgloss.Color
	ShowBorder  bool
	// Individual border control
	BorderTop    bool
	BorderBottom bool
	BorderLeft   bool
	BorderRight  bool
	// Progress bar support
	ProgressBar   progress.Model
	ProgressPercent float64
	ShowProgress  bool
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

// CalculateBorderReduction returns the width and height reduction needed for borders
func (p *Pane) CalculateBorderReduction() (widthReduction, heightReduction int) {
	if !p.ShowBorder {
		return 0, 0
	}
	
	if p.BorderLeft {
		widthReduction++
	}
	if p.BorderRight {
		widthReduction++
	}
	if p.BorderTop {
		heightReduction++
	}
	if p.BorderBottom {
		heightReduction++
	}
	
	return widthReduction, heightReduction
}

// Pane methods
func (p *Pane) Render(width, height int) string {
	if !p.Visible {
		return ""
	}

	p.Width = width
	p.Height = height

	// Calculate viewport size accounting for actual borders
	widthReduction, heightReduction := p.CalculateBorderReduction()
	viewportWidth := width - widthReduction
	viewportHeight := height - heightReduction
	if viewportWidth < 0 {
		viewportWidth = 0
	}
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	if p.Viewport.Width != viewportWidth || p.Viewport.Height != viewportHeight {
		p.Viewport.Width = viewportWidth
		p.Viewport.Height = viewportHeight
	}

	// Use custom render function if available
	if p.RenderFunc != nil {
		// If progress bar is enabled, we need to integrate it into custom render
		if p.ShowProgress {
			log.Printf("DEBUG: Custom RenderFunc exists for pane %s with ShowProgress=true, progress bar may not render", p.ID)
		}
		return p.RenderFunc(p, width, height)
	}

	// Default rendering
	var content string
	
	// Render progress bar if enabled
	if p.ShowProgress {
		log.Printf("DEBUG: Entering progress bar rendering for pane %s", p.ID)
		// Calculate available width for progress bar (accounting for borders)
		progressWidth := viewportWidth
		if progressWidth < 10 {
			progressWidth = 10 // Minimum width
		}
		if progressWidth > 80 {
			progressWidth = 80 // Maximum width for readability
		}
		
		// Update progress bar width if needed
		if p.ProgressBar.Width != progressWidth {
			p.ProgressBar.Width = progressWidth
		}
		
		// Render progress bar
		progressContent := p.ProgressBar.ViewAs(p.ProgressPercent)
		
		// Debug: Log progress bar rendering
		log.Printf("DEBUG: Rendering progress bar in pane %s: ShowProgress=%v, Width=%d, Percent=%f, Content length=%d, Content=%q", 
			p.ID, p.ShowProgress, p.ProgressBar.Width, p.ProgressPercent, len(progressContent), progressContent)
		
		// If progress bar content is empty or just whitespace, something is wrong
		if strings.TrimSpace(progressContent) == "" {
			// Fallback: create a simple text representation
			progressContent = fmt.Sprintf("[%s] %.0f%%", strings.Repeat(" ", int(p.ProgressBar.Width/2)), p.ProgressPercent*100)
			log.Printf("DEBUG: Progress bar content was empty, using fallback: %q", progressContent)
		}
		
		// Add some padding/spacing around the progress bar
		pad := strings.Repeat(" ", 2)
		progressWithPadding := pad + progressContent + "\n"
		
		// Combine with other content if present
		// Priority: Progress bar should always be visible when ShowProgress is true
		// When progress bar is enabled, we need to ensure it's always visible
		// The issue is that viewport content might scroll and hide the progress bar
		// So we'll render progress bar separately and reduce viewport height to make room
		
		// Calculate how much space the progress bar needs (roughly 3 lines: label + bar + padding)
		progressBarHeight := 3
		
		// If viewport exists and has dimensions, we need to adjust its height to make room for progress bar
		if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
			// Temporarily reduce viewport height to make room for progress bar
			originalViewportHeight := p.Viewport.Height
			adjustedViewportHeight := viewportHeight - progressBarHeight
			if adjustedViewportHeight < 1 {
				adjustedViewportHeight = 1
			}
			
			// Set adjusted height temporarily
			p.Viewport.Height = adjustedViewportHeight
			viewportContent := p.Viewport.View()
			// Restore original height
			p.Viewport.Height = originalViewportHeight
			
			// Check if viewport actually has content (not just empty lines)
			if strings.TrimSpace(viewportContent) != "" {
				// Always show progress bar first, then viewport content
				// Add a label to make progress bar more visible
				label := fmt.Sprintf("Progress: %.0f%%", p.ProgressPercent*100)
				progressSection := "\n" + pad + label + "\n" + progressWithPadding
				
				if p.Title != "" {
					titleWidth := width - widthReduction
					if titleWidth < 0 {
						titleWidth = 0
					}
					titleBar := p.Style.Copy().
						Width(titleWidth).
						Border(lipgloss.NormalBorder()).
						BorderBottom(false).
						Render(p.Title)
					// Progress bar goes after title, before viewport
					content = lipgloss.JoinVertical(lipgloss.Top, titleBar, progressSection, viewportContent)
				} else {
					// Progress bar first, then viewport
					content = lipgloss.JoinVertical(lipgloss.Top, progressSection, viewportContent)
				}
			} else {
				// Viewport has no content, just show progress bar
				if p.Title != "" {
					titleWidth := width - widthReduction
					if titleWidth < 0 {
						titleWidth = 0
					}
					titleBar := p.Style.Copy().
						Width(titleWidth).
						Border(lipgloss.NormalBorder()).
						BorderBottom(false).
						Render(p.Title)
					label := fmt.Sprintf("Progress: %.0f%%", p.ProgressPercent*100)
					content = lipgloss.JoinVertical(lipgloss.Top, titleBar, "\n"+pad+label+"\n", progressWithPadding)
				} else {
					label := fmt.Sprintf("Progress: %.0f%%", p.ProgressPercent*100)
					content = "\n" + pad + label + "\n" + progressWithPadding
				}
			}
		} else if p.Content != "" {
			// Progress bar first, then static content
			content = lipgloss.JoinVertical(lipgloss.Top, progressWithPadding, p.Content)
		} else {
			// Just show progress bar with some vertical padding
			// Add a label to make it more visible for debugging
			label := fmt.Sprintf("Progress: %.0f%%", p.ProgressPercent*100)
			content = "\n" + pad + label + "\n" + progressWithPadding
		}
		log.Printf("DEBUG: Progress bar rendering complete for pane %s, content length=%d", p.ID, len(content))
	} else if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
		// Get viewport content
		viewportContent := p.Viewport.View()
		
		if p.Title != "" {
			// Title bar width needs to account for borders if present
			titleWidth := width - widthReduction
			if titleWidth < 0 {
				titleWidth = 0
			}
			titleBar := p.Style.Copy().
				Width(titleWidth).
				Border(lipgloss.NormalBorder()).
				BorderBottom(false).
				Render(p.Title)
			// Join title with viewport content
			content = lipgloss.JoinVertical(lipgloss.Top, titleBar, viewportContent)
		} else {
			content = viewportContent
		}
	} else {
		content = p.Content
	}

	// Apply border styling if enabled
	if p.ShowBorder {
		// Content is already sized to viewportWidth x viewportHeight
		// Render with border at width x height - lipgloss will fit the content correctly
		style := p.Style.Copy().
			Width(width).
			Height(height).
			BorderStyle(p.BorderStyle)
		
		// Apply individual border settings
		style = style.BorderTop(p.BorderTop).
			BorderBottom(p.BorderBottom).
			BorderLeft(p.BorderLeft).
			BorderRight(p.BorderRight)
		
		if p.BorderColor != "" {
			style = style.BorderForeground(p.BorderColor)
		}
		
		return style.Render(content)
	}

	return p.Style.Copy().Width(width).Height(height).Render(content)
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
	
	// Calculate viewport size accounting for actual borders
	widthReduction, heightReduction := p.CalculateBorderReduction()
	viewportWidth := width - widthReduction
	viewportHeight := height - heightReduction
	if viewportWidth < 0 {
		viewportWidth = 0
	}
	if viewportHeight < 0 {
		viewportHeight = 0
	}
	
	if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
		p.Viewport.Width = viewportWidth
		p.Viewport.Height = viewportHeight
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
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Window size is handled at the layout level
		if p.ShowProgress {
			// Update progress bar width on window resize
			p.ProgressBar.Width = p.Width - 4 // Account for borders
			if p.ProgressBar.Width < 10 {
				p.ProgressBar.Width = 10
			}
			if p.ProgressBar.Width > 80 {
				p.ProgressBar.Width = 80
			}
		}
	case tea.KeyMsg:
		// Pass key messages to viewport for scrolling
		if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
			p.Viewport, cmd = p.Viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case tea.MouseMsg:
		// Mouse events are handled at the layout level, but we can pass them through
		// for viewport scrolling (e.g., mouse wheel)
		if p.Viewport.Width > 0 && p.Viewport.Height > 0 {
			p.Viewport, cmd = p.Viewport.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case progress.FrameMsg:
		// Handle progress bar animation frames
		if p.ShowProgress {
			var progressCmd tea.Cmd
			progressModel, progressCmd := p.ProgressBar.Update(msg)
			p.ProgressBar = progressModel.(progress.Model)
			if progressCmd != nil {
				cmds = append(cmds, progressCmd)
			}
		}
	}
	if len(cmds) > 0 {
		return p, tea.Batch(cmds...)
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
		ID:          id,
		Type:        paneType,
		Viewport:    vp,
		Content:     "",
		Visible:     true,
		Style:       lipgloss.NewStyle(),
		MinWidth:    10,
		MinHeight:   5,
		BorderStyle: lipgloss.NormalBorder(),
		BorderColor: "",
		ShowBorder:  true,
		// Default borders: bottom, left, right (no top border)
		BorderTop:    false,
		BorderBottom: true,
		BorderLeft:   true,
		BorderRight:  true,
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

// SetBorderStyle sets the border style for a pane
func (p *Pane) SetBorderStyle(borderType string, color string) {
	borderType = strings.ToLower(strings.TrimSpace(borderType))
	switch borderType {
	case "normal":
		p.BorderStyle = lipgloss.NormalBorder()
	case "rounded":
		p.BorderStyle = lipgloss.RoundedBorder()
	case "thick":
		p.BorderStyle = lipgloss.ThickBorder()
	case "double":
		p.BorderStyle = lipgloss.DoubleBorder()
	case "hidden":
		p.BorderStyle = lipgloss.HiddenBorder()
		p.ShowBorder = false
	default:
		p.BorderStyle = lipgloss.NormalBorder()
	}
	
	if color != "" {
		p.BorderColor = lipgloss.Color(color)
	}
}

