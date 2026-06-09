//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "testing"

// TestOptionItems confirms a select option renders its label plus its optional
// description, and that the value is never shown as the description — so a value
// that encodes data (like "host url") stays out of the list.
func TestOptionItems(t *testing.T) {
	opts := []Option{
		{Label: "With desc", Value: "v1", Description: "a description"},
		{Label: "No desc", Value: "some long encoded value"},
		{Label: "Empty value", Description: "only a description"},
	}
	items := optionItems(opts)

	if items[0].name != "With desc" || items[0].desc != "a description" {
		t.Fatalf("item0 = %+v, want label/desc to match", items[0])
	}
	if items[1].name != "No desc" || items[1].desc != "" {
		t.Fatalf("item1 = %+v, want blank desc (value must not appear)", items[1])
	}
	if items[2].name != "Empty value" || items[2].desc != "only a description" {
		t.Fatalf("item2 = %+v", items[2])
	}
}
