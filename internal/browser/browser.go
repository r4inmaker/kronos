package browser

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
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
	treeMu     *sync.Mutex
}

func NewBrowser(ctx context.Context, logger *logger.Logger) *Browser {
	return &Browser{
		Context:    ctx,
		Logger:     logger,
		Nodes:      make([]*accessibility.Node, 0),
		NodeMap:    make(map[accessibility.NodeID]*accessibility.Node),
		FilterFunc: FilterNodeDefault(),
		treeMu:     &sync.Mutex{},
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

// Send keys
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
		_, exp, err := runtime.CallFunctionOn(`function() { this.focus(); this.select()}`).
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

		// Undecorate
		b.UndecorateInteractable(ctx, node)

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
			const color = "#31b0ff";
			const bg = "rgba(0, 220, 254, 0.33)";

			// 1. Kill all potential "active" rings from the browser
			this.style.setProperty('outline', 'none', 'important');
			this.style.setProperty('border-color', 'transparent', 'important');

			// 2. Use a spread shadow to simulate a 2px border
			// Syntax: x-offset y-offset blur spread color
			// This won't change the size of your input box at all.
			//this.style.setProperty('box-shadow', '0 0 0 2px ' + color, 'important');
			
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

// UndecorateInteractable removes the decoration styles applied by DecorateInteractable
func (b *Browser) UndecorateInteractable(ctx context.Context, node *accessibility.Node) error {
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
			// Remove the styles we added
			this.style.removeProperty('outline');
			this.style.removeProperty('border-color');
			this.style.removeProperty('box-shadow');
			this.style.removeProperty('background-color');
			this.style.removeProperty('color');
			this.style.removeProperty('visibility');
		} catch (e) {}
	}`).WithObjectID(obj.ObjectID).Do(ctx)

	if err != nil {
		return err
	}
	if exception != nil {
		return fmt.Errorf("undecoration exception: %s", exception.Exception.Description)
	}

	return nil
}

// Logo
func (b *Browser) InjectLogo() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Read the SVG file from project root
		svgData, err := os.ReadFile("img/logo.svg")
		if err != nil {
			// Try alternative path (in case working directory is different)
			svgData, err = os.ReadFile("../../img/logo.svg")
			if err != nil {
				b.Logger.Debug(fmt.Sprintf("Could not read logo file: %v", err))
				return nil // Don't fail if logo missing, just skip
			}
		}

		// Encode SVG as base64 data URL
		logoBase64 := base64.StdEncoding.EncodeToString(svgData)
		dataURL := "data:image/svg+xml;base64," + logoBase64

		// JavaScript with underscore blinker
		jsCode := fmt.Sprintf(`
			(function() {
				if (document.getElementById('kronos-logo-container')) {
					return;
				}
				
				// Create main container
				const container = document.createElement('div');
				container.id = 'kronos-logo-container';
				
				// Create inner wrapper for text and logo
				const wrapper = document.createElement('div');
				wrapper.id = 'kronos-wrapper';
				
				// Add styles
				const style = document.createElement('style');
				style.textContent = 
					"#kronos-logo-container {" +
					"position: fixed !important;" +
					"top: 20px !important;" +
					"right: 20px !important;" +
					"z-index: 999999 !important;" +
					"background: rgba(0, 0, 0, 0.85) !important;" +
					"border-radius: 12px !important;" +
					"backdrop-filter: blur(10px) !important;" +
					"padding: 12px 16px !important;" +
					"border: 1px solid rgba(37, 118, 176, 0.4) !important;" +
					"box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;" +
					"}" +
					"#kronos-wrapper {" +
					"display: flex !important;" +
					"align-items: center !important;" +
					"gap: 12px !important;" +
					"flex-direction: row !important;" +
					"}" +
					"#kronos-logo {" +
					"width: 36px !important;" +
					"height: 36px !important;" +
					"display: block !important;" +
					"order: 2 !important;" +
					"}" +
					"#kronos-text {" +
					"color: #6bb5e8 !important;" +
					"font-family: 'Courier New', monospace !important;" +
					"font-size: 13px !important;" +
					"font-weight: 500 !important;" +
					"order: 1 !important;" +
					"}" +
					"#kronos-blinker {" +
					"display: inline-block !important;" +
					"color: #6bb5e8 !important;" +
					"font-family: 'Courier New', monospace !important;" +
					"font-size: 13px !important;" +
					"font-weight: 500 !important;" +
					"animation: kronosBlink 1s step-end infinite !important;" +
					"}" +
					"@keyframes kronosBlink {" +
					"0%%, 100%% { opacity: 1; }" +
					"50%% { opacity: 0; }" +
					"}";
				
				document.head.appendChild(style);
				
				// Create logo image
				const logo = document.createElement('img');
				logo.id = 'kronos-logo';
				logo.src = '%s';
				logo.alt = 'Kronos Logo';
				logo.title = 'Kronos Automation';
				
				// Create text container
				const textContainer = document.createElement('div');
				textContainer.id = 'kronos-text';
				
				// Full message to type
				const fullMessage = "Your browser is being controlled by Kronos";
				let index = 0;
				
				// Create span for the typed text
				const textSpan = document.createElement('span');
				textContainer.appendChild(textSpan);
				
				// Create underscore blinker
				const blinker = document.createElement('span');
				blinker.id = 'kronos-blinker';
				blinker.textContent = '_';
				textContainer.appendChild(blinker);
				
				// Typewriter effect
				function typeWriter() {
					if (index < fullMessage.length) {
						textSpan.textContent += fullMessage.charAt(index);
						index++;
						setTimeout(typeWriter, 100);
					}
				}
				
				// Start the typewriter effect
				typeWriter();
				
				// Assemble everything
				wrapper.appendChild(textContainer);
				wrapper.appendChild(logo);
				container.appendChild(wrapper);
				document.body.appendChild(container);
			})();
		`, dataURL)

		_, _, err = runtime.Evaluate(jsCode).Do(ctx)
		if err != nil {
			b.Logger.Debug(fmt.Sprintf("Failed to inject logo: %v", err))
			return err
		}

		//b.Logger.Info("Logo injected successfully")
		return nil
	})
}

// Utility Functions
func (b *Browser) SuppressConsole() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, _, err := runtime.Evaluate(`
            console.log = function(){};
            console.warn = function(){};
            console.error = function(){};
        `).Do(ctx)
		return err
	})
}
