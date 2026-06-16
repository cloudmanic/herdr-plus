//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "github.com/charmbracelet/lipgloss"

// Palette. A small, cohesive set of colors for a clean dark-terminal look,
// shared by the fuzzyList component and the projects browser in this package.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(lipgloss.Color("#A78BFA")).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	countStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	nameStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	nameSelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	matchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F2A900")).Bold(true)
	barStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	footerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563"))
	headingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Bold(true)
)
