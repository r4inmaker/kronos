package browser

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/r4inmaker/kronos/internal/logger"
)

type Browser struct {
	Context    context.Context
	Logger     *logger.Logger
	Nodes      []*accessibility.Node
	NodeMap    map[accessibility.NodeID]*accessibility.Node
	FilterFunc nodeFilterFunc
	Root       *accessibility.Node
}

func NewBrowser(ctx context.Context, logger *logger.Logger) *Browser {
	return &Browser{
		Context:    ctx,
		Logger:     logger,
		Nodes:      make([]*accessibility.Node, 0),
		NodeMap:    make(map[accessibility.NodeID]*accessibility.Node),
		FilterFunc: FilterNodeDefault(),
	}
}

func (b *Browser) Execute(actions ...chromedp.Action) error {
	var throttledActions []chromedp.Action
	if len(actions) == 1 {
		return chromedp.Run(b.Context, actions[0])
	}

	for _, action := range actions {
		throttledActions = append(throttledActions, action)
		throttledActions = append(throttledActions, chromedp.Sleep(200 * time.Millisecond))
	}

	return chromedp.Run(b.Context, throttledActions...)
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

// Wait
func (b *Browser) WaitForLifecycle(eventName string, timeout time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// enable lifecycle events
		if err := page.Enable().Do(ctx); err != nil {
			return err
		}
		if err := page.SetLifecycleEventsEnabled(true).Do(ctx); err != nil {
			return err
		}

		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		done := make(chan struct{})

		chromedp.ListenTarget(cctx, func(ev interface{}) {
			if e, ok := ev.(*page.EventLifecycleEvent); ok {
				if e.Name == eventName {
					select {
					case <-done:
					default:
						close(done)
					}
				}
			}
		})

		select {
		case <-done:
			return nil
		case <-cctx.Done():
			return fmt.Errorf("timeout waiting for %s", eventName)
		}
	})
}
