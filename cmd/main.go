package main

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/r4inmaker/kronos/internal/browser"
)




func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-infobars", true),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()
	//ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	//defer cancel()
	b := browser.NewBrowser(ctx)
	b.Execute(
		b.Navigate("https://linkedin.com"),
		b.WaitReady("body"),
	)	
	b.BuildTree()
	b.BuildFilteredTree(
		browser.FilterNodeAND(
				browser.FilterNodeByName("sign in"),
				browser.FilterNodeByRole("link"),
			),
	)
	

	fmt.Println(b.SprintTree(true))

	

	for _, node := range b.FilteredNodeMap {
		if browser.FilterNodeByName("sign in")(node) {
			b.Execute(
				b.Click(browser.SelectorFromNode(node)),
			)		
		}
	}
}