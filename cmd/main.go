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
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()
	//ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	//defer cancel()
	b := browser.NewBrowser(ctx)
	b.Execute(
		chromedp.Navigate("https://linkedin.com"),
		chromedp.WaitReady("body"),
	)	
	b.BuildTree()
	b.BuildFilteredTree(
		browser.FilterNodeAND(
				browser.FilterNodeByName("sign in"),
				browser.FilterNodeByRole("link"),
			),)

	fmt.Println(b.SprintTree(true))
}