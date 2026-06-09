//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "testing"

// TestOptionItems confirms select options render their label plus optional
// description (never the raw value), that separators become non-selectable
// heading/spacer rows, and that selectable rows keep their original options
// index in ref even when separators are interspersed.
func TestOptionItems(t *testing.T) {
	opts := []Option{
		{Label: "With desc", Value: "v1", Description: "a description"}, // 0
		{Label: "No desc", Value: "some long encoded value"},            // 1
		{Heading: "Group B"},                  // 2: heading separator
		{Label: "After heading", Value: "v3"}, // 3
		{},                                    // 4: blank spacer
	}
	items := optionItems(opts)

	if len(items) != 5 {
		t.Fatalf("got %d items, want 5", len(items))
	}
	if !items[0].selectable || items[0].name != "With desc" || items[0].desc != "a description" || items[0].ref != 0 {
		t.Fatalf("item0 = %+v", items[0])
	}
	if !items[1].selectable || items[1].name != "No desc" || items[1].desc != "" || items[1].ref != 1 {
		t.Fatalf("item1 = %+v, value must not appear as desc", items[1])
	}
	if items[2].selectable || items[2].name != "Group B" {
		t.Fatalf("item2 = %+v, want non-selectable heading", items[2])
	}
	if !items[3].selectable || items[3].name != "After heading" || items[3].ref != 3 {
		t.Fatalf("item3 = %+v, ref must be original options index 3", items[3])
	}
	if items[4].selectable || items[4].name != "" {
		t.Fatalf("item4 = %+v, want blank spacer", items[4])
	}
}
