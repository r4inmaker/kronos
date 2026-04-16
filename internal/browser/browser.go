package browser

import (
	"context"
	"fmt"
	"math/rand"
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
	return chromedp.Run(b.Context, actions...)
}

// Navigate
func (b *Browser) Navigate(url string) chromedp.Action {
	return chromedp.Navigate(url)
}

// Click
func (b *Browser) ClickNode(id int64) chromedp.Action {
	nodeID := accessibility.NodeID(strconv.Itoa(int(id)))
	return chromedp.ActionFunc(func(ctx context.Context) error {
		node, ok := b.NodeMap[nodeID]
		if !ok {
			return fmt.Errorf("node %d not found in tree", id)
		}
		b.DecorateInteractable(ctx, node)

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

func (b *Browser) SendKeysNode(id int64, keys string) chromedp.Action {
	nodeID := accessibility.NodeID(strconv.Itoa(int(id)))

	return chromedp.ActionFunc(func(ctx context.Context) error {
		node, ok := b.NodeMap[nodeID]
		if !ok {
			return fmt.Errorf("node %d not found in tree", id)
		}

		// 1. Resolve the accessibility/tree node to an object ID
		obj, err := dom.ResolveNode().WithBackendNodeID(node.BackendDOMNodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("could not resolve backend node: %v", err)
		}

		// 2. Focus the element via JS (essential for the Input domain to target correctly)
		_, exp, err := runtime.CallFunctionOn(`function() { this.focus(); }`).
			WithObjectID(obj.ObjectID).Do(ctx)
		if err != nil || exp != nil {
			return fmt.Errorf("focus failed: %v", err)
		}

		// Decorate
		b.DecorateInteractable(ctx, node)

		// 3. Iterate and type like a human
		for _, char := range keys {
			// Dispatch KeyEvent for each character
			err := chromedp.KeyEvent(string(char)).Do(ctx)
			if err != nil {
				return err
			}

			// Add jitter/randomness (100ms - 250ms)
			delay := time.Duration(50+rand.Intn(50)) * time.Millisecond
			time.Sleep(delay)
		}

		return nil
	})
}

// Wait
func (b *Browser) WaitReady(sel string) chromedp.Action {
	return chromedp.WaitReady(sel)
}
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

func (b *Browser) WaitSleep(t time.Duration) chromedp.Action {
	return chromedp.Sleep(t)
}

// DecorateInteractable now runs synchronously within the provided context.
func (b *Browser) DecorateInteractable(ctx context.Context, node *accessibility.Node) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}

	// Resolve the node to an ObjectID
	obj, err := dom.ResolveNode().WithBackendNodeID(node.BackendDOMNodeID).Do(ctx)
	if err != nil {
		return err
	}

	_, exception, err := runtime.CallFunctionOn(`function() {
		try {
			const color = "#2576b0";
			const bg = "rgba(37, 118, 176, 0.12)";

			// 1. Kill all potential "active" rings from the browser
			this.style.setProperty('outline', 'none', 'important');
			this.style.setProperty('border-color', 'transparent', 'important');

			// 2. Use a spread shadow to simulate a 2px border
			// Syntax: x-offset y-offset blur spread color
			// This won't change the size of your input box at all.
			this.style.setProperty('box-shadow', '0 0 0 2px ' + color, 'important');
			
			// 3. Set Background and Text
			this.style.setProperty('background-color', bg, 'important');
			this.style.setProperty('color', color, 'important');

			// 4. Ensure it's visible and focused
			this.style.setProperty('visibility', 'visible', 'important');
			
			this.scrollIntoView({behavior: "instant", block: "center"});
			this.focus();
		} catch (e) {}
	}`).WithObjectID(obj.ObjectID).Do(ctx)

	if err != nil {
		return err
	}
	// Correct way to check the exception details
	if exception != nil {
		return fmt.Errorf("decoration exception: %s", exception.Exception.Description)
	}

	return nil
}
