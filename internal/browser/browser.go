package browser

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/r4inmaker/kronos/internal/logger"
)

type Browser struct {
	Context         context.Context
	Logger 		    *logger.Logger
	Nodes           []*accessibility.Node
	NodeMap         map[accessibility.NodeID]*accessibility.Node
	FilteredNodeMap map[accessibility.NodeID]*accessibility.Node
    FilterFunc      nodeFilterFunc
	Root		    *accessibility.Node
}

func NewBrowser(ctx context.Context, logger *logger.Logger) *Browser {
	return &Browser{
		Context:         ctx,
		Logger: 				 logger,
		Nodes:           make([]*accessibility.Node, 0),
		NodeMap:         make(map[accessibility.NodeID]*accessibility.Node),
		FilteredNodeMap: make(map[accessibility.NodeID]*accessibility.Node),
        FilterFunc:      FilterNodeInteractable(),
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

// Click
func (b *Browser) ClickNode(id int64) chromedp.Action {
    nodeID := accessibility.NodeID(strconv.Itoa(int(id)))
    return chromedp.ActionFunc(func(ctx context.Context) error {
        node, ok := b.NodeMap[nodeID]
        if !ok {
            return fmt.Errorf("node %d not found in tree", id)
        }

        // Primary: BackendDOMNodeID
        obj, err := dom.ResolveNode().WithBackendNodeID(node.BackendDOMNodeID).Do(ctx)
        if err == nil {
            _, exceptionDetails, err := runtime.CallFunctionOn(`function() {
                this.scrollIntoView({behavior: "instant", block: "center"});
                this.click();
            }`).
                WithObjectID(obj.ObjectID).
                Do(ctx)
            if err == nil && exceptionDetails == nil {
								b.Logger.Info(fmt.Sprintf(`Clicked node: "%s" %s [%s]`,
								GetNodeName(node), GetNodeRole(node), GetNodeID(node)))
                return nil
            }
            b.Logger.Debug(fmt.Sprintf("primary click failed (%v), falling back to XPath", err))
        } else {
			b.Logger.Debug(fmt.Sprintf("primary click failed (%v), falling back to XPath", err))
        }

        // Fallback: XPath
        xpath := SelectorFromNode(node)
        if xpath == "" {
            return fmt.Errorf("primary click failed and no xpath fallback available")
        }
        
        return chromedp.Click(xpath, chromedp.BySearch).Do(ctx)
    })
}

func (b *Browser) SendKeysNode(id int64, keys string, simulate ...bool) chromedp.Action {
    nodeID := accessibility.NodeID(strconv.Itoa(int(id)))
    forceSimulate := len(simulate) > 0 && simulate[0]
	return chromedp.ActionFunc(func(ctx context.Context) error {
		node, ok := b.NodeMap[nodeID]
		if !ok {
			return fmt.Errorf("node %d not found in tree", id)
		}

        if !forceSimulate {
            // Primary: BackendDOMNodeID + javascript 
            obj, err := dom.ResolveNode().WithBackendNodeID(node.BackendDOMNodeID).Do(ctx)
            if err == nil {
                _, exceptionDetails, err := runtime.CallFunctionOn(`function(keys) {
                    this.scrollIntoView({behavior: "instant", block: "center"});
                    this.focus();
                    this.value = keys;
                    this.dispatchEvent(new Event('input', { bubbles: true }));
                    this.dispatchEvent(new Event('change', { bubbles: true }));
                }`).
                    WithObjectID(obj.ObjectID).
                    WithArguments([]*runtime.CallArgument{
                        {Value: []byte(fmt.Sprintf(`%q`, keys))},
                    }).
                    Do(ctx)
                if err == nil && exceptionDetails == nil {
                    b.Logger.Info(fmt.Sprintf(`Sent keys to node: "%s" %s [%s]`,
                        GetNodeName(node), GetNodeRole(node), GetNodeID(node)))
                    return nil
                }
                b.Logger.Debug(fmt.Sprintf("primary sendkeys failed (%v), falling back to XPath", err))
            } else {
                b.Logger.Debug(fmt.Sprintf("primary sendkeys failed (%v), falling back to XPath", err))
            }
        }


		// Fallback: XPath
		xpath := SelectorFromNode(node)
		if xpath == "" {
			return fmt.Errorf("primary sendkeys failed and no xpath fallback available")
		}
		return chromedp.SendKeys(xpath, keys, chromedp.BySearch).Do(ctx)
	})
}