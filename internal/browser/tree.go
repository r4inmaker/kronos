package browser

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/chromedp"
)

// Building the tree
func (b *Browser) BuildTree() error {
	b.FilteredNodeMap = make(map[accessibility.NodeID]*accessibility.Node)

	// (Re)create nodes
	if err := b.Execute(
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			b.Nodes, err = accessibility.GetFullAXTree().Do(ctx)
			return err
		}),
	); err != nil {
		return err
	}

	// (Re)create a map
	b.NodeMap = make(map[accessibility.NodeID]*accessibility.Node)
	for _, n := range b.Nodes {
		b.NodeMap[n.NodeID] = n
	}

	// Find the root
	b.Root = b.FindRoot()

	return nil
}

func (b *Browser) BuildFilteredTree() error {
	filteredNodeMap := make(map[accessibility.NodeID]*accessibility.Node)
	// Rebuild the full tree first
	if err := b.BuildTree(); err != nil {
		return err
	}

	// Build the filtered tree
	var recurse func(node *accessibility.Node) bool
	recurse = func(node *accessibility.Node) bool {
		matched := false

		// Recursively check children first
		for _, childNodeID := range node.ChildIDs {
			if childNode, ok := b.NodeMap[childNodeID]; ok {
				if recurse(childNode) {
					matched = true
				}
			}
		}

		// Check the node itself
		if b.FilterFunc(node) {
			matched = true
		}

		// Append node once if matched
		if matched {
			if _, exists := filteredNodeMap[node.NodeID]; !exists {
				filteredNodeMap[node.NodeID] = node
			}
		}

		return matched
	}

	recurse(b.Root)
	b.FilteredNodeMap = filteredNodeMap
	return nil
}


func (b *Browser) FindRoot() *accessibility.Node {
    // Build a set of all child IDs
    childIDs := make(map[accessibility.NodeID]bool)
    for _, n := range b.Nodes {
        for _, cid := range n.ChildIDs {
            childIDs[cid] = true
        }
    }
    // Root is the node that is nobody's child
    for _, n := range b.Nodes {
        if !childIDs[n.NodeID] {
            return n
        }
    }
    return b.Nodes[0] // fallback
}

func (b *Browser) SprintTree(filter nodeFilterFunc) string {
    if b.Root == nil {
        return ""
    }
    var sb strings.Builder
    sprintTree(b.Root.NodeID, b.NodeMap, &sb, 0, filter)
    return sb.String()
}

func sprintTree(id accessibility.NodeID, nodeMap map[accessibility.NodeID]*accessibility.Node,
    builder *strings.Builder, depth int, filter nodeFilterFunc) {
    node, ok := nodeMap[id]
    if !ok {
        return
    }

    // only print if this node itself matches the filter
    if filter == nil || filter(node) {
        indent := strings.Repeat("  ", depth)
        role := GetNodeRole(node)
        name := GetNodeName(node)
        if name == "" {
            builder.WriteString(fmt.Sprintf("%s%s [%s]\n", indent, role, id))
        } else {
            builder.WriteString(fmt.Sprintf("%s%s %q [%s]\n", indent, role, name, id))
        }
    }

    for _, childID := range node.ChildIDs {
        sprintTree(childID, nodeMap, builder, depth+1, filter)
    }
}

// Filter functions
type nodeFilterFunc func(node *accessibility.Node) bool

func FilterNodeByRole(roles ...string) nodeFilterFunc {
	return func(node *accessibility.Node) bool {
		if len(roles) == 0 {
			return true
		}
		if node.Role == nil {
			return false
		}
		nodeRole := GetNodeRole(node)
		for _, r := range roles {
			if nodeRole == r {
				return true
			}
		}
		return false
	}
}

func FilterNodeByName(names ...string) nodeFilterFunc {
	return func(node *accessibility.Node) bool {
		if len(names) == 0 {
			return true
		}
		if node.Name == nil {
			return false
		}
		nodeName := GetNodeName(node)
		for _, n := range names {
			if strings.EqualFold(nodeName, n) {
				return true
			}
		}
		return false
	}
}

func FilterNodeAND(filterFuncs ...nodeFilterFunc) nodeFilterFunc {
	return func(node *accessibility.Node) bool {
		for _, f := range filterFuncs {
			if !f(node) {
				return false
			}
		}
		return true
	}
}

func FilterNodeOR(filterFuncs ...nodeFilterFunc) nodeFilterFunc {
	return func(node *accessibility.Node) bool {
		for _, f := range filterFuncs {
			if f(node) {
				return true
			}
		}
		return false
	}
}

func FilterNodeInteractable() nodeFilterFunc {
    return FilterNodeByRole(
        "button",
        "link",
        "textbox",
        "checkbox",
        "radio",
        "combobox",
        "menuitem",
        "tab",
        "heading",
        "alert",
        "dialog",
        "main",
        "form",
        "RootWebArea",
    )
}

// Selector functions
func SelectorFromNode(node *accessibility.Node) string {
	role := GetNodeRole(node)
	name := GetNodeName(node)

	switch role {
	case "button":
		return fmt.Sprintf(`//button[normalize-space()="%s"]`, name)
	case "link":
		return fmt.Sprintf(`//a[normalize-space()="%s"]`, name)
	case "textbox":
		return fmt.Sprintf(`//input[@placeholder="%s" or @aria-label="%s"]`, name, name)
	default:
		return fmt.Sprintf(`//*[@aria-label="%s"]`, name)
	}
}

// Utility functions
func GetNodeName(node *accessibility.Node) string {
	if node.Name == nil {
		return ""
	}
	return strings.Trim(string(node.Name.Value), `"`)
}

func GetNodeRole(node *accessibility.Node) string {
	if node.Role == nil {
		return ""
	}
	return strings.Trim(string(node.Role.Value), `"`)
}

func GetNodeID(node *accessibility.Node) string {
	return strings.Trim(string(node.NodeID), `"`)
}
