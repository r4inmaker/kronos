package browser

import (
	"context"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/chromedp"
)
type Browser struct {
	Context 				  context.Context
	Nodes 		[]*accessibility.Node
	NodeMap				map[accessibility.NodeID]*accessibility.Node
	FilteredNodeMap				map[accessibility.NodeID]*accessibility.Node
}

func NewBrowser(ctx context.Context) *Browser {
	return &Browser{
		Context: ctx,
		Nodes: make([]*accessibility.Node, 0),
		NodeMap: make(map[accessibility.NodeID]*accessibility.Node),
		FilteredNodeMap: make(map[accessibility.NodeID]*accessibility.Node),
	}
}

func (b *Browser) Execute(actions... chromedp.Action) error {
	return chromedp.Run(b.Context, actions...)
}


