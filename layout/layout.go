package layout

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// LayoutCommandMsg is sent to request layout operations
// Note: Session is passed as interface{} to avoid import cycle
type LayoutCommandMsg struct {
	Command string
	Args    []string
	Session interface{} // *session.Session, but we can't import it here
}

// Layout manages the overall pane layout
type Layout struct {
	Root       Node
	ActivePane string // ID of the currently active pane
	Width      int
	Height     int
	// Drag state tracking
	DraggingPane string // ID of pane being dragged, empty if not dragging
	DragStartX   int    // Initial mouse X position
	DragStartY   int    // Initial mouse Y position
	DragCurrentX int    // Current mouse X position
	DragCurrentY int    // Current mouse Y position
	// Resize state tracking
	ResizingContainer *Container // Container being resized, nil if not resizing
	ResizeStartX      int        // Initial mouse X when resize started
	ResizeStartY      int        // Initial mouse Y when resize started
	ResizeStartSplit  int        // Initial split percentage when resize started
	// Pane positions for hit testing (updated during render)
	PanePositions map[string]PanePosition
	// Container positions for split boundary detection
	ContainerPositions map[*Container]ContainerPosition
}

// PanePosition stores the absolute screen position of a pane
type PanePosition struct {
	X      int
	Y      int
	Width  int
	Height int
}

// ContainerPosition stores the absolute screen position and split boundary of a container
type ContainerPosition struct {
	X           int
	Y           int
	Width       int
	Height      int
	SplitX      int // X position of split boundary (for horizontal splits)
	SplitY      int // Y position of split boundary (for vertical splits)
	Direction   SplitDirection
}

// NewLayout creates a new layout with a single main viewport
func NewLayout(mainPaneID string) *Layout {
	mainPane := NewPane(mainPaneID, PaneTypeViewport)
	// Main pane doesn't have borders by default
	mainPane.ShowBorder = false
	return &Layout{
		Root:               mainPane,
		ActivePane:         mainPaneID,
		PanePositions:      make(map[string]PanePosition),
		ContainerPositions: make(map[*Container]ContainerPosition),
	}
}

// Render renders the entire layout
func (l *Layout) Render() string {
	if l.Root == nil {
		return "No layout"
	}
	// Update pane and container positions before rendering for hit testing
	l.updatePanePositions()
	l.updateContainerPositions()
	return l.Root.Render(l.Width, l.Height)
}

// SetSize sets the size of the layout
func (l *Layout) SetSize(width, height int) {
	l.Width = width
	l.Height = height
	if l.Root != nil {
		l.Root.SetSize(width, height)
	}
}

// FindPane finds a pane by ID
func (l *Layout) FindPane(id string) *Pane {
	if l.Root == nil {
		return nil
	}
	return l.Root.FindPane(id)
}

// GetActivePane returns the currently active pane
func (l *Layout) GetActivePane() *Pane {
	return l.FindPane(l.ActivePane)
}

// GetAllPanes returns all panes in the layout
func (l *Layout) GetAllPanes() []*Pane {
	if l.Root == nil {
		return nil
	}
	return l.Root.GetAllPanes()
}

// Split splits a pane in the specified direction
// Returns an error if the pane is not found or cannot be split
func (l *Layout) Split(paneID string, direction SplitDirection, splitPercent int, newPaneID string, newPaneType PaneType) error {
	if splitPercent < 5 || splitPercent > 95 {
		return fmt.Errorf("split percentage must be between 5 and 95")
	}

	pane := l.FindPane(paneID)
	if pane == nil {
		return fmt.Errorf("pane %s not found", paneID)
	}

	// Check if new pane ID already exists
	if l.FindPane(newPaneID) != nil {
		return fmt.Errorf("pane %s already exists", newPaneID)
	}

	// Create new pane
	newPane := NewPane(newPaneID, newPaneType)

	// Replace the pane with a container
	container := NewContainer(direction, splitPercent, pane, newPane)
	l.Root = l.replaceNode(l.Root, paneID, container)

	return nil
}

// Unsplit removes a pane and merges its space with its sibling
// If removing the last pane in a container, the container is removed
func (l *Layout) Unsplit(paneID string) error {
	if l.Root == nil {
		return fmt.Errorf("layout is empty")
	}

	// Can't remove the root if it's the only pane
	if pane, ok := l.Root.(*Pane); ok {
		if pane.ID == paneID {
			return fmt.Errorf("cannot remove the last pane")
		}
	}

	// Find and remove the pane
	newRoot, err := l.removeNode(l.Root, paneID)
	if err != nil {
		return err
	}

	l.Root = newRoot

	// If the active pane was removed, switch to another pane
	if l.ActivePane == paneID {
		panes := l.GetAllPanes()
		if len(panes) > 0 {
			l.ActivePane = panes[0].ID
		}
	}

	return nil
}

// replaceNode replaces a node in the tree
func (l *Layout) replaceNode(node Node, targetID string, replacement Node) Node {
	if pane, ok := node.(*Pane); ok {
		if pane.ID == targetID {
			return replacement
		}
		return pane
	}

	if container, ok := node.(*Container); ok {
		// Check if target is in left or right subtree
		if container.Left.FindPane(targetID) != nil {
			container.Left = l.replaceNode(container.Left, targetID, replacement)
		} else if container.Right.FindPane(targetID) != nil {
			container.Right = l.replaceNode(container.Right, targetID, replacement)
		}
		return container
	}

	return node
}

// removeNode removes a node from the tree
func (l *Layout) removeNode(node Node, targetID string) (Node, error) {
	if pane, ok := node.(*Pane); ok {
		if pane.ID == targetID {
			return nil, fmt.Errorf("cannot remove root pane")
		}
		return pane, nil
	}

	if container, ok := node.(*Container); ok {
		// Check if target is in left subtree
		if container.Left.FindPane(targetID) != nil {
			if pane, ok := container.Left.(*Pane); ok && pane.ID == targetID {
				// Remove left, return right
				return container.Right, nil
			}
			// Recursively remove from left
			newLeft, err := l.removeNode(container.Left, targetID)
			if err != nil {
				return nil, err
			}
			container.Left = newLeft
			return container, nil
		}

		// Check if target is in right subtree
		if container.Right.FindPane(targetID) != nil {
			if pane, ok := container.Right.(*Pane); ok && pane.ID == targetID {
				// Remove right, return left
				return container.Left, nil
			}
			// Recursively remove from right
			newRight, err := l.removeNode(container.Right, targetID)
			if err != nil {
				return nil, err
			}
			container.Right = newRight
			return container, nil
		}
	}

	return node, nil
}

// SetActivePane sets the active pane
func (l *Layout) SetActivePane(paneID string) error {
	if l.FindPane(paneID) == nil {
		return fmt.Errorf("pane %s not found", paneID)
	}
	l.ActivePane = paneID
	return nil
}

// Update passes messages to the layout tree
func (l *Layout) Update(msg tea.Msg) tea.Cmd {
	if l.Root == nil {
		return nil
	}

	// Handle mouse events for dragging
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		return l.handleMouseEvent(mouseMsg)
	}

	var cmd tea.Cmd
	l.Root, cmd = l.Root.Update(msg)
	return cmd
}

// handleMouseEvent processes mouse events for pane dragging and resizing
func (l *Layout) handleMouseEvent(msg tea.MouseMsg) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// First check if clicking near a split boundary (resize mode)
			container := l.findContainerAtSplitBoundary(msg.X, msg.Y)
			if container != nil {
				l.ResizingContainer = container
				l.ResizeStartX = msg.X
				l.ResizeStartY = msg.Y
				l.ResizeStartSplit = container.Split
			} else {
				// Otherwise, check for pane dragging
				paneID := l.findPaneAtPosition(msg.X, msg.Y)
				if paneID != "" {
					l.DraggingPane = paneID
					l.DragStartX = msg.X
					l.DragStartY = msg.Y
					l.DragCurrentX = msg.X
					l.DragCurrentY = msg.Y
				}
			}
		}
	case tea.MouseActionMotion:
		if l.ResizingContainer != nil {
			// Resize mode: adjust split percentage
			l.adjustSplitForResize(msg.X, msg.Y)
		} else if l.DraggingPane != "" {
			// Drag mode: track position for potential swap
			l.DragCurrentX = msg.X
			l.DragCurrentY = msg.Y
		}
	case tea.MouseActionRelease:
		if msg.Button == tea.MouseButtonLeft {
			if l.ResizingContainer != nil {
				// Finalize resize
				l.ResizingContainer = nil
			} else if l.DraggingPane != "" {
				// Finalize drag - check if we should swap panes
				targetPaneID := l.findPaneAtPosition(msg.X, msg.Y)
				if targetPaneID != "" && targetPaneID != l.DraggingPane {
					// Swap the panes
					l.swapPanes(l.DraggingPane, targetPaneID)
				}
				// Reset drag state
				l.DraggingPane = ""
			}
		}
	}
	return nil
}

// findPaneAtPosition finds the pane at the given screen coordinates
func (l *Layout) findPaneAtPosition(x, y int) string {
	// Update pane positions first
	l.updatePanePositions()
	
	// Check each pane's position
	for paneID, pos := range l.PanePositions {
		if x >= pos.X && x < pos.X+pos.Width &&
			y >= pos.Y && y < pos.Y+pos.Height {
			return paneID
		}
	}
	return ""
}

// updatePanePositions calculates and stores the absolute position of each pane
func (l *Layout) updatePanePositions() {
	l.PanePositions = make(map[string]PanePosition)
	if l.Root != nil {
		l.calculateNodePositions(l.Root, 0, 0, l.Width, l.Height)
	}
}

// updateContainerPositions calculates and stores the absolute position of each container and its split boundary
func (l *Layout) updateContainerPositions() {
	l.ContainerPositions = make(map[*Container]ContainerPosition)
	if l.Root != nil {
		l.calculateContainerPositions(l.Root, 0, 0, l.Width, l.Height)
	}
}

// calculateContainerPositions recursively calculates positions of all containers
func (l *Layout) calculateContainerPositions(node Node, x, y, width, height int) {
	if _, ok := node.(*Pane); ok {
		// Pane - no container to track
		return
	}

	if container, ok := node.(*Container); ok {
		splitPos := container.Split
		if splitPos < 0 {
			splitPos = 0
		}
		if splitPos > 100 {
			splitPos = 100
		}

		pos := ContainerPosition{
			X:         x,
			Y:         y,
			Width:     width,
			Height:    height,
			Direction: container.Direction,
		}

		if container.Direction == SplitHorizontal {
			// Split left/right
			leftWidth := (width * splitPos) / 100
			rightWidth := width - leftWidth

			// Ensure minimum sizes
			leftMin := container.Left.GetMinWidth()
			rightMin := container.Right.GetMinWidth()

			if leftWidth < leftMin {
				leftWidth = leftMin
				rightWidth = width - leftWidth
			}
			if rightWidth < rightMin {
				rightWidth = rightMin
				leftWidth = width - rightWidth
			}

			pos.SplitX = x + leftWidth

			l.ContainerPositions[container] = pos

			// Recurse into children
			l.calculateContainerPositions(container.Left, x, y, leftWidth, height)
			l.calculateContainerPositions(container.Right, x+leftWidth, y, rightWidth, height)
		} else {
			// Split top/bottom
			topHeight := (height * splitPos) / 100
			bottomHeight := height - topHeight

			// Ensure minimum sizes
			topMin := container.Left.GetMinHeight()
			bottomMin := container.Right.GetMinHeight()

			if topHeight < topMin {
				topHeight = topMin
				bottomHeight = height - topHeight
			}
			if bottomHeight < bottomMin {
				bottomHeight = bottomMin
				topHeight = height - bottomHeight
			}

			pos.SplitY = y + topHeight

			l.ContainerPositions[container] = pos

			// Recurse into children
			l.calculateContainerPositions(container.Left, x, y, width, topHeight)
			l.calculateContainerPositions(container.Right, x, y+topHeight, width, bottomHeight)
		}
	}
}

// findContainerAtSplitBoundary finds a container whose split boundary is near the given coordinates
// Returns nil if no split boundary is nearby (within threshold pixels)
func (l *Layout) findContainerAtSplitBoundary(x, y int) *Container {
	const threshold = 2 // pixels from boundary to trigger resize (makes it easier to grab)
	
	// Update container positions first
	l.updateContainerPositions()

	for container, pos := range l.ContainerPositions {
		if container.Direction == SplitHorizontal {
			// Check if near vertical split boundary
			if y >= pos.Y && y < pos.Y+pos.Height {
				if x >= pos.SplitX-threshold && x <= pos.SplitX+threshold {
					return container
				}
			}
		} else {
			// Check if near horizontal split boundary
			if x >= pos.X && x < pos.X+pos.Width {
				if y >= pos.SplitY-threshold && y <= pos.SplitY+threshold {
					return container
				}
			}
		}
	}
	return nil
}

// adjustSplitForResize adjusts the split percentage based on mouse movement
func (l *Layout) adjustSplitForResize(x, y int) {
	if l.ResizingContainer == nil {
		return
	}

	container := l.ResizingContainer
	pos, ok := l.ContainerPositions[container]
	if !ok {
		return
	}

	var newSplit int

	if container.Direction == SplitHorizontal {
		// Horizontal split: adjust based on X movement
		deltaX := x - l.ResizeStartX
		// Convert pixel movement to percentage change
		// 1 pixel = (1/width) * 100 percentage points
		if pos.Width > 0 {
			percentChange := (deltaX * 100) / pos.Width
			newSplit = l.ResizeStartSplit + percentChange
		} else {
			newSplit = l.ResizeStartSplit
		}
	} else {
		// Vertical split: adjust based on Y movement
		deltaY := y - l.ResizeStartY
		// Convert pixel movement to percentage change
		// 1 pixel = (1/height) * 100 percentage points
		if pos.Height > 0 {
			percentChange := (deltaY * 100) / pos.Height
			newSplit = l.ResizeStartSplit + percentChange
		} else {
			newSplit = l.ResizeStartSplit
		}
	}

	// Clamp to valid range (5-95)
	if newSplit < 5 {
		newSplit = 5
	}
	if newSplit > 95 {
		newSplit = 95
	}

	// Check minimum sizes
	if container.Direction == SplitHorizontal {
		leftMin := container.Left.GetMinWidth()
		rightMin := container.Right.GetMinWidth()
		minLeftPercent := (leftMin * 100) / pos.Width
		maxLeftPercent := 100 - ((rightMin * 100) / pos.Width)
		if newSplit < minLeftPercent {
			newSplit = minLeftPercent
		}
		if newSplit > maxLeftPercent {
			newSplit = maxLeftPercent
		}
	} else {
		topMin := container.Left.GetMinHeight()
		bottomMin := container.Right.GetMinHeight()
		minTopPercent := (topMin * 100) / pos.Height
		maxTopPercent := 100 - ((bottomMin * 100) / pos.Height)
		if newSplit < minTopPercent {
			newSplit = minTopPercent
		}
		if newSplit > maxTopPercent {
			newSplit = maxTopPercent
		}
	}

	container.Split = newSplit
	
	// Update container positions after resize so next render is accurate
	l.updateContainerPositions()
}

// calculateNodePositions recursively calculates positions of all panes
func (l *Layout) calculateNodePositions(node Node, x, y, width, height int) {
	if pane, ok := node.(*Pane); ok {
		l.PanePositions[pane.ID] = PanePosition{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		}
		return
	}

	if container, ok := node.(*Container); ok {
		splitPos := container.Split
		if splitPos < 0 {
			splitPos = 0
		}
		if splitPos > 100 {
			splitPos = 100
		}

		if container.Direction == SplitHorizontal {
			// Split left/right
			leftWidth := (width * splitPos) / 100
			rightWidth := width - leftWidth

			// Ensure minimum sizes
			leftMin := container.Left.GetMinWidth()
			rightMin := container.Right.GetMinWidth()

			if leftWidth < leftMin {
				leftWidth = leftMin
				rightWidth = width - leftWidth
			}
			if rightWidth < rightMin {
				rightWidth = rightMin
				leftWidth = width - rightWidth
			}

			l.calculateNodePositions(container.Left, x, y, leftWidth, height)
			l.calculateNodePositions(container.Right, x+leftWidth, y, rightWidth, height)
		} else {
			// Split top/bottom
			topHeight := (height * splitPos) / 100
			bottomHeight := height - topHeight

			// Ensure minimum sizes
			topMin := container.Left.GetMinHeight()
			bottomMin := container.Right.GetMinHeight()

			if topHeight < topMin {
				topHeight = topMin
				bottomHeight = height - topHeight
			}
			if bottomHeight < bottomMin {
				bottomHeight = bottomMin
				topHeight = height - bottomHeight
			}

			l.calculateNodePositions(container.Left, x, y, width, topHeight)
			l.calculateNodePositions(container.Right, x, y+topHeight, width, bottomHeight)
		}
	}
}

// swapPanes swaps two panes in the layout tree
func (l *Layout) swapPanes(paneID1, paneID2 string) {
	pane1 := l.FindPane(paneID1)
	pane2 := l.FindPane(paneID2)
	
	if pane1 == nil || pane2 == nil {
		return
	}

	// Simple swap: exchange the panes' positions in the tree
	// This is a simplified implementation - a full implementation would
	// need to handle the tree structure more carefully
	// For now, we'll just swap their content and properties as a placeholder
	// A proper implementation would require restructuring the tree
	
	// Store temporary values
	tempContent := pane1.Content
	tempViewport := pane1.Viewport
	tempType := pane1.Type
	tempTitle := pane1.Title
	tempStyle := pane1.Style
	tempBorderStyle := pane1.BorderStyle
	tempBorderColor := pane1.BorderColor
	tempShowBorder := pane1.ShowBorder

	// Swap pane1 -> pane2
	pane1.Content = pane2.Content
	pane1.Viewport = pane2.Viewport
	pane1.Type = pane2.Type
	pane1.Title = pane2.Title
	pane1.Style = pane2.Style
	pane1.BorderStyle = pane2.BorderStyle
	pane1.BorderColor = pane2.BorderColor
	pane1.ShowBorder = pane2.ShowBorder

	// Swap pane2 -> pane1
	pane2.Content = tempContent
	pane2.Viewport = tempViewport
	pane2.Type = tempType
	pane2.Title = tempTitle
	pane2.Style = tempStyle
	pane2.BorderStyle = tempBorderStyle
	pane2.BorderColor = tempBorderColor
	pane2.ShowBorder = tempShowBorder
}

// AdjustSplit adjusts the split percentage of a container
func (l *Layout) AdjustSplit(containerID string, delta int) error {
	// Find the container (we need to traverse to find containers)
	// For now, we'll adjust splits by finding containers that contain specific panes
	// This is a simplified version - you might want to add container IDs
	return nil
}

// ListPanes returns a formatted list of all panes
func (l *Layout) ListPanes() string {
	panes := l.GetAllPanes()
	if len(panes) == 0 {
		return "No panes"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%-15s %-15s %-10s %-10s", "ID", "Type", "Width", "Height"))
	lines = append(lines, strings.Repeat("-", 50))

	for _, pane := range panes {
		active := ""
		if pane.ID == l.ActivePane {
			active = "*"
		}
		lines = append(lines, fmt.Sprintf("%s%-14s %-15s %-10d %-10d",
			active, pane.ID, string(pane.Type), pane.Width, pane.Height))
	}

	return strings.Join(lines, "\n")
}

// GetPaneInfo returns information about a specific pane
func (l *Layout) GetPaneInfo(paneID string) string {
	pane := l.FindPane(paneID)
	if pane == nil {
		return fmt.Sprintf("Pane %s not found", paneID)
	}

	active := ""
	if pane.ID == l.ActivePane {
		active = " (active)"
	}

	return fmt.Sprintf("Pane: %s%s\nType: %s\nSize: %dx%d\nVisible: %v\nTitle: %s",
		pane.ID, active, pane.Type, pane.Width, pane.Height, pane.Visible, pane.Title)
}

