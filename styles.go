//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "github.com/charmbracelet/lipgloss"

// Palette. A small, cohesive set of colors shared by the fuzzyList component
// and the projects browser in this package. Foregrounds that render directly
// against the terminal background use ANSI palette colors (or the terminal's
// default foreground) instead of fixed hex values, so text stays readable on
// both light and dark terminal themes. titleStyle keeps its hex pair because
// it sets both foreground and background, making it self-contained.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(lipgloss.Color("#A78BFA")).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	countStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	nameStyle    = lipgloss.NewStyle()
	nameSelStyle = lipgloss.NewStyle().Bold(true)
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	matchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	barStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	footerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)
