package browser

import (
	"context"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/chromedp"
)

type Browser struct {
	Context         context.Context
	Nodes           []*accessibility.Node
	NodeMap         map[accessibility.NodeID]*accessibility.Node
	FilteredNodeMap map[accessibility.NodeID]*accessibility.Node
}

func NewBrowser(ctx context.Context) *Browser {
	return &Browser{
		Context:         ctx,
		Nodes:           make([]*accessibility.Node, 0),
		NodeMap:         make(map[accessibility.NodeID]*accessibility.Node),
		FilteredNodeMap: make(map[accessibility.NodeID]*accessibility.Node),
	}
}

func (b *Browser) Execute(actions ...chromedp.Action) error {
	return chromedp.Run(b.Context, actions...)
}

func (b *Browser) Navigate(url string) chromedp.Action {
	return chromedp.Navigate(url)
}

func (b *Browser) WaitReady(sel string) chromedp.Action {
	return chromedp.WaitReady(sel)
}

func (b *Browser) Click(identifier any) chromedp.Action {
	// Find a node either by id or name
	return chromedp.Click(identifier)
}
