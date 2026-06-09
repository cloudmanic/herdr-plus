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

// TestActionListItems confirms project and global actions are grouped under
// "Project"/"Global" headings when both are present — with each selectable row's
// ref pointing into the concatenated project++global order — and that a
// global-only set renders as a plain, ungrouped list (no headings), preserving
// the pre-feature look for repos without a .herdr-plus directory.
func TestActionListItems(t *testing.T) {
	project := []Action{
		{Name: "make build", Description: "Build", origin: originProject},
		{Name: "make test", Description: "Test", origin: originProject},
	}
	global := []Action{
		{Name: "GitHub", Description: "Open GitHub", origin: originGlobal},
	}

	items := actionListItems(project, global)
	// Project heading, 2 project rows, Global heading, 1 global row.
	if len(items) != 5 {
		t.Fatalf("got %d items, want 5: %+v", len(items), items)
	}
	if items[0].selectable || items[0].name != "Project" {
		t.Fatalf("item0 = %+v, want non-selectable Project heading", items[0])
	}
	if !items[1].selectable || items[1].name != "make build" || items[1].ref != 0 {
		t.Fatalf("item1 = %+v, want make build with ref 0", items[1])
	}
	if !items[2].selectable || items[2].name != "make test" || items[2].ref != 1 {
		t.Fatalf("item2 = %+v, want make test with ref 1", items[2])
	}
	if items[3].selectable || items[3].name != "Global" {
		t.Fatalf("item3 = %+v, want non-selectable Global heading", items[3])
	}
	if !items[4].selectable || items[4].name != "GitHub" || items[4].ref != 2 {
		t.Fatalf("item4 = %+v, want GitHub with ref 2 (index into project++global)", items[4])
	}

	// Global-only: no headings, refs start at 0.
	only := actionListItems(nil, global)
	if len(only) != 1 {
		t.Fatalf("got %d items for global-only, want 1 (no headings): %+v", len(only), only)
	}
	if !only[0].selectable || only[0].name != "GitHub" || only[0].ref != 0 {
		t.Fatalf("global-only item = %+v, want GitHub ref 0 with no heading", only[0])
	}
}
