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

	return nil
}

func (b *Browser) BuildFilteredTree(filterFunc nodeFilterFunc) {
	filteredNodeMap := make(map[accessibility.NodeID]*accessibility.Node)

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
		if filterFunc(node) {
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

	recurse(b.Nodes[0])
	b.FilteredNodeMap = filteredNodeMap
}

// Printing the tree for the LLM
func (b *Browser) SprintTree(filtered bool) string {
	var sb strings.Builder
	var nodeMap map[accessibility.NodeID]*accessibility.Node
	switch filtered {
		case true:nodeMap = b.FilteredNodeMap
		default: nodeMap = b.NodeMap
	}
	sprintTree(b.Nodes[0].NodeID, nodeMap, &sb, 0)
	return sb.String()
}

func sprintTree(id accessibility.NodeID, nodeMap map[accessibility.NodeID]*accessibility.Node, 
	builder *strings.Builder, depth int) {
	node, ok := nodeMap[id]
	if !ok {
		return
	}

	indent := strings.Repeat("  ", depth)
	role := getNodeRole(node)
	name := getNodeName(node)


	switch name {
	case "": builder.WriteString(fmt.Sprintf("%s%s [%s]\n", indent, role, id))
	default: builder.WriteString(fmt.Sprintf("%s%s %q [%s]\n", indent, role, name, id))
	}
	

	for _, childID := range node.ChildIDs {
		sprintTree(childID, nodeMap, builder, depth+1)
	}
}

// Node filtering (focus only on relevant nodes)
type nodeFilterFunc func(node *accessibility.Node) bool

func FilterNodeByRole(roles... string) nodeFilterFunc {
		return func(node *accessibility.Node) bool {
			if len(roles) == 0 {return true}
			if node.Role == nil {return false}
				nodeRole := strings.Trim(string(node.Role.Value), `"`)
				for _, r := range roles {
					if nodeRole == r {
						return true
					}
				}
			return false
		}
}

func FilterNodeByName(names... string) nodeFilterFunc {
		return func(node *accessibility.Node) bool {
			if len(names) == 0 {return true}
			if node.Name == nil {return false}
			nodeName := strings.Trim(string(node.Name.Value), `"`)
			for _, n := range names {
				if strings.EqualFold(nodeName, n) {
					return true
				}
			}
			return false
	}
}

func FilterNodeAND(filterFuncs... nodeFilterFunc) nodeFilterFunc {
	return func(node *accessibility.Node) bool {
		for _, f := range filterFuncs {
				if !f(node) {
					return false
				}
			}
			return true
	}
}

func FilterNodeOR(filterFuncs... nodeFilterFunc) nodeFilterFunc {
	return func(node *accessibility.Node) bool {
		for _, f := range filterFuncs {
				if f(node) {
					return true
				}
			}
			return false
	}
}

func FilterNodeDefault() nodeFilterFunc {
	return func(node *accessibility.Node) bool {
		return true
	}
}

// XPath Selector
func SelectorFromNode(node *accessibility.Node) string {
    role := strings.Trim(string(node.Role.Value), `"`)
    name := strings.Trim(string(node.Name.Value), `"`)

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
func getNodeName(node *accessibility.Node) string {
	if node.Name == nil {
		return ""
	}
	return strings.Trim(string(node.Name.Value), `"`)
}

func getNodeRole(node *accessibility.Node) string {
	if node.Role == nil {
		return ""
	}
	return strings.Trim(string(node.Role.Value), `"`)
}