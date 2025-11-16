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
}

// NewLayout creates a new layout with a single main viewport
func NewLayout(mainPaneID string) *Layout {
	mainPane := NewPane(mainPaneID, PaneTypeViewport)
	return &Layout{
		Root:       mainPane,
		ActivePane: mainPaneID,
	}
}

// Render renders the entire layout
func (l *Layout) Render() string {
	if l.Root == nil {
		return "No layout"
	}
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

	var cmd tea.Cmd
	l.Root, cmd = l.Root.Update(msg)
	return cmd
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

