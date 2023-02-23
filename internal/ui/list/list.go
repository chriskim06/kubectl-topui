package list

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
)

// Item is an item that appears in the list.
type Item interface {
	// FilterValue is the value we use when filtering against this item when
	// we're filtering the list.
	FilterValue() string
}

// ItemDelegate encapsulates the general functionality for all list items. The
// benefit to separating this logic from the item itself is that you can change
// the functionality of items without changing the actual items themselves.
//
// Note that if the delegate also implements help.KeyMap delegate-related
// help items will be added to the help view.
type ItemDelegate interface {
	// Render renders the item's view.
	Render(w io.Writer, m Model, index int, item Item)

	// Height is the height of the list item.
	Height() int

	// Spacing is the size of the horizontal gap between list items in cells.
	Spacing() int

	// Update is the update loop for items. All messages in the list's update
	// loop will pass through here except when the user is setting a filter.
	// Use this method to perform item-level updates appropriate to this
	// delegate.
	Update(msg tea.Msg, m *Model) tea.Cmd
}

// Model contains the state of this component.
type Model struct {
	showTitle      bool
	showPagination bool

	itemNameSingular string
	itemNamePlural   string

	Title  string
	Styles Styles

	// Key mappings for navigating the list.
	KeyMap KeyMap

	width     int
	height    int
	Paginator paginator.Model
	cursor    int
	offset    int

	statusMessage      string
	statusMessageTimer *time.Timer

	// The master set of items we're working with.
	items []Item

	delegate ItemDelegate
}

// New returns a new model with sensible defaults.
func New(items []Item, delegate ItemDelegate, width, height int) Model {
	styles := DefaultStyles()

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = styles.ActivePaginationDot.String()
	p.InactiveDot = styles.InactivePaginationDot.String()

	m := Model{
		showTitle:        true,
		showPagination:   true,
		itemNameSingular: "item",
		itemNamePlural:   "items",
		KeyMap:           DefaultKeyMap(),
		Styles:           styles,
		Title:            "List",

		width:     width,
		height:    height,
		delegate:  delegate,
		items:     items,
		Paginator: p,
	}

	m.updatePagination()
	return m
}

func (m Model) GetOffset() int {
	return m.offset
}

// SetShowTitle shows or hides the title bar.
func (m *Model) SetShowTitle(v bool) {
	m.showTitle = v
	m.updatePagination()
}

// ShowTitle returns whether or not the title bar is set to be rendered.
func (m Model) ShowTitle() bool {
	return m.showTitle
}

// SetShowPagination hides or shows the paginator. Note that pagination will
// still be active, it simply won't be displayed.
func (m *Model) SetShowPagination(v bool) {
	m.showPagination = v
	m.updatePagination()
}

// ShowPagination returns whether the pagination is visible.
func (m *Model) ShowPagination() bool {
	return m.showPagination
}

// Items returns the items in the list.
func (m Model) Items() []Item {
	return m.items
}

// SetItems sets the items available in the list. This returns a command.
func (m *Model) SetItems(i []Item) tea.Cmd {
	m.items = i
	m.updatePagination()
	return nil
}

// SetDelegate sets the item delegate.
func (m *Model) SetDelegate(d ItemDelegate) {
	m.delegate = d
	m.updatePagination()
}

// VisibleItems returns the total items available to be shown.
func (m Model) VisibleItems() []Item {
	return m.items
}

// SelectedItem returns the current selected item in the list.
func (m Model) SelectedItem() Item {
	i := m.Index()

	items := m.VisibleItems()
	if i < 0 || len(items) == 0 || len(items) <= i {
		return nil
	}

	return items[i]
}

// Index returns the index of the currently selected item as it appears in the
// entire slice of items.
func (m Model) Index() int {
	return m.Paginator.Page*m.Paginator.PerPage + m.cursor
}

// Cursor returns the index of the cursor on the current page.
func (m Model) Cursor() int {
	return m.cursor
}

// CursorUp moves the cursor up. This can also move the state to the previous
// page.
func (m *Model) CursorUp() {
	m.cursor--

	// If we're at the start, stop
	if m.cursor < 0 && m.Paginator.Page == 0 {
		m.cursor = 0
		return
	}

	// Move the cursor as normal
	if m.cursor >= 0 {
		return
	}

	// Go to the previous page
	m.Paginator.PrevPage()
	m.cursor = m.Paginator.ItemsOnPage(len(m.VisibleItems())) - 1
}

// CursorDown moves the cursor down. This can also advance the state to the
// next page.
func (m *Model) CursorDown() {
	itemsOnPage := m.Paginator.ItemsOnPage(len(m.VisibleItems()))

	m.cursor++

	// If we're at the end, stop
	if m.cursor < itemsOnPage {
		return
	}

	// Go to the next page
	if !m.Paginator.OnLastPage() {
		m.Paginator.NextPage()
		m.cursor = 0
		return
	}

	// During filtering the cursor position can exceed the number of
	// itemsOnPage. It's more intuitive to start the cursor at the
	// topmost position when moving it down in this scenario.
	if m.cursor > itemsOnPage {
		m.cursor = 0
		return
	}

	m.cursor = itemsOnPage - 1
}

func (m *Model) CursorRight() {
	m.offset++
}

func (m *Model) CursorLeft() {
	if m.offset == 0 {
		return
	}
	m.offset--
}

// Width returns the current width setting.
func (m Model) Width() int {
	return m.width
}

// Height returns the current height setting.
func (m Model) Height() int {
	return m.height
}

// SetSize sets the width and height of this component.
func (m *Model) SetSize(width, height int) {
	m.setSize(width, height)
}

// SetWidth sets the width of this component.
func (m *Model) SetWidth(v int) {
	m.setSize(v, m.height)
}

// SetHeight sets the height of this component.
func (m *Model) SetHeight(v int) {
	m.setSize(m.width, v)
}

func (m *Model) setSize(width, height int) {
	m.width = width
	m.height = height
	m.updatePagination()
}

// Update pagination according to the amount of items for the current state.
func (m *Model) updatePagination() {
	index := m.Index()
	offset := m.GetOffset()
	availHeight := m.height

	if m.showTitle {
		availHeight -= lipgloss.Height(m.titleView())
	}
	if m.showPagination {
		availHeight -= lipgloss.Height(m.paginationView())
	}

	m.Paginator.PerPage = max(1, availHeight/(m.delegate.Height()+m.delegate.Spacing()))

	if pages := len(m.VisibleItems()); pages < 1 {
		m.Paginator.SetTotalPages(1)
	} else {
		m.Paginator.SetTotalPages(pages)
	}

	// Restore index
	m.Paginator.Page = index / m.Paginator.PerPage
	m.cursor = index % m.Paginator.PerPage
	m.offset = offset

	// Make sure the page stays in bounds
	if m.Paginator.Page >= m.Paginator.TotalPages-1 {
		m.Paginator.Page = max(0, m.Paginator.TotalPages-1)
	}
}

// Update is the Bubble Tea update loop.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.KeyMap.ForceQuit) {
			return m, tea.Quit
		}
	}

	cmds = append(cmds, m.handleBrowsing(msg))
	return m, tea.Batch(cmds...)
}

// Updates for when a user is browsing the list.
func (m *Model) handleBrowsing(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	numItems := len(m.VisibleItems())

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			return tea.Quit

		case key.Matches(msg, m.KeyMap.CursorUp):
			m.CursorUp()

		case key.Matches(msg, m.KeyMap.CursorDown):
			m.CursorDown()

		case key.Matches(msg, m.KeyMap.CursorLeft):
			m.CursorLeft()

		case key.Matches(msg, m.KeyMap.CursorRight):
			m.CursorRight()

		case key.Matches(msg, m.KeyMap.PrevPage):
			m.Paginator.PrevPage()

		case key.Matches(msg, m.KeyMap.NextPage):
			m.Paginator.NextPage()

		case key.Matches(msg, m.KeyMap.GoToStart):
			m.Paginator.Page = 0
			m.cursor = 0
			m.offset = 0

		case key.Matches(msg, m.KeyMap.GoToEnd):
			m.Paginator.Page = m.Paginator.TotalPages - 1
			m.cursor = m.Paginator.ItemsOnPage(numItems) - 1
			m.offset = 0
		}
	}

	// Keep the index in bounds when paginating
	itemsOnPage := m.Paginator.ItemsOnPage(len(m.VisibleItems()))
	if m.cursor > itemsOnPage-1 {
		m.cursor = max(0, itemsOnPage-1)
	}

	return tea.Batch(cmds...)
}

// View renders the component.
func (m Model) View() string {
	var (
		sections    []string
		availHeight = m.height
	)

	if m.showTitle {
		v := m.titleView()
		sections = append(sections, v)
		availHeight -= lipgloss.Height(v)
	}

	var pagination string
	if m.showPagination {
		pagination = m.paginationView()
		availHeight -= lipgloss.Height(pagination)
	}

	content := lipgloss.NewStyle().Height(availHeight).Render(m.populatedView())
	sections = append(sections, content)

	if m.showPagination {
		sections = append(sections, pagination)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) titleView() string {
	var (
		view          string
		titleBarStyle = m.Styles.TitleBar.Copy()
	)

	// draw the title.
	if m.showTitle {
		titleStr := m.Title
		if m.offset >= len(titleStr) {
			titleStr = ""
		} else {
			titleStr = titleStr[m.offset:]
		}
		titleStr = truncate.StringWithTail(titleStr, uint(m.width), "…")
		view += m.Styles.Title.Render(titleStr)
	}

	if len(view) > 0 {
		return titleBarStyle.Render(view)
	}
	return view
}

func (m Model) paginationView() string {
	if m.Paginator.TotalPages < 2 { //nolint:gomnd
		return ""
	}

	s := m.Paginator.View()

	// If the dot pagination is wider than the width of the window
	// use the arabic paginator.
	if ansi.PrintableRuneWidth(s) > m.width {
		m.Paginator.Type = paginator.Arabic
		s = m.Styles.ArabicPagination.Render(m.Paginator.View())
	}

	style := m.Styles.PaginationStyle
	if m.delegate.Spacing() == 0 && style.GetMarginTop() == 0 {
		style = style.Copy().MarginTop(1)
	}

	return style.Render(s)
}

func (m Model) populatedView() string {
	items := m.VisibleItems()

	var b strings.Builder

	// Empty states
	if len(items) == 0 {
		return m.Styles.NoItems.Render("No " + m.itemNamePlural + " found.")
	} else {
		start, end := m.Paginator.GetSliceBounds(len(items))
		docs := items[start:end]

		for i, item := range docs {
			m.delegate.Render(&b, m, i+start, item)
			if i != len(docs)-1 {
				fmt.Fprint(&b, strings.Repeat("\n", m.delegate.Spacing()+1))
			}
		}
	}

	// If there aren't enough items to fill up this page (always the last page)
	// then we need to add some newlines to fill up the space where items would
	// have been.
	itemsOnPage := m.Paginator.ItemsOnPage(len(items))
	if itemsOnPage < m.Paginator.PerPage {
		n := (m.Paginator.PerPage - itemsOnPage) * (m.delegate.Height() + m.delegate.Spacing())
		if len(items) == 0 {
			n -= m.delegate.Height() - 1
		}
		fmt.Fprint(&b, strings.Repeat("\n", n))
	}

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
