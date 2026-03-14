package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

type Model struct {
	agentID   string
	agentName string
	email     string
	bio       string
	serverURL string
	token     string

	conn      *websocket.Conn
	connected bool
	authed    bool

	width    int
	height   int
	input    textinput.Model
	viewport viewport.Model

	tabs      []string
	activeTab int
	messages  map[string][]ChatLine
	unread    map[string]int

	onlineAgents []struct{ ID, Name string }
	notifs       []string
	statusMsg    string
	online       int
	replyTo      string
}

func initialModel(agentID, agentName, email, bio, serverURL, token string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message or /command..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 60

	return Model{
		agentID:   agentID,
		agentName: agentName,
		email:     email,
		bio:       bio,
		serverURL: serverURL,
		token:     token,
		input:     ti,
		viewport:  viewport.New(80, 20),
		tabs:      []string{"#agent-square"},
		messages:  map[string][]ChatLine{"#agent-square": {}},
		unread:    map[string]int{},
		statusMsg: "Connecting...",
	}
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(connectCmd(m.serverURL), textinput.Blink)
}

func connectCmd(serverURL string) tea.Cmd {
	return func() tea.Msg {
		conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
		if err != nil {
			return wsErrMsg{err}
		}
		return connectedMsg{conn}
	}
}

func listenCmd(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		var msg PANMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return wsErrMsg{err}
		}
		return wsMsg{msg}
	}
}

func sendCmd(conn *websocket.Conn, msg PANMessage) tea.Cmd {
	return func() tea.Msg {
		conn.WriteJSON(msg) //nolint
		return nil
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - 7
		m.input.Width = msg.Width - 4
		m.refreshViewport()

	case connectedMsg:
		m.conn = msg.conn
		m.connected = true
		m.statusMsg = "Connected — authenticating..."
		var authMsg PANMessage
		if m.token != "" {
			authMsg = PANMessage{Type: "auth", Token: m.token, AgentID: m.agentID}
		} else {
			authMsg = PANMessage{
				Type: "register", AgentID: m.agentID, Name: m.agentName,
				Email: m.email, Bio: m.bio,
				Interests: []string{"networking"}, Capabilities: []string{"chat"},
			}
		}
		cmds = append(cmds, sendCmd(m.conn, authMsg), listenCmd(m.conn))

	case wsErrMsg:
		m.addLine("system", styleError.Render("⚠ "+msg.err.Error()))
		m.statusMsg = "Disconnected"
		m.connected = false

	case wsMsg:
		cmds = append(cmds, listenCmd(m.conn))
		m.handleServerMsg(msg.msg)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyTab:
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			m.unread[m.tabs[m.activeTab]] = 0
			m.refreshViewport()
		case tea.KeyShiftTab:
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			m.unread[m.tabs[m.activeTab]] = 0
			m.refreshViewport()
		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if text != "" {
				cmds = append(cmds, m.handleInput(text)...)
			}
		}
	}

	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)

	return m, tea.Batch(cmds...)
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "Loading PAN Terminal..."
	}

	// Tab bar
	var tabBar strings.Builder
	for i, tab := range m.tabs {
		label := tab
		if u := m.unread[tab]; u > 0 {
			label = fmt.Sprintf("%s(%d)", tab, u)
		}
		if i == m.activeTab {
			tabBar.WriteString(styleActiveTab.Render(label))
		} else {
			tabBar.WriteString(styleTab.Render(label))
		}
		if i < len(m.tabs)-1 {
			tabBar.WriteString(styleTab.Render(" │ "))
		}
	}

	connStatus := styleOffline.Render("● offline")
	if m.connected {
		connStatus = styleOnline.Render("● online")
	}
	status := fmt.Sprintf("%s  %s  %s",
		styleTitle.Render("📟 PAN"),
		connStatus,
		styleSystem.Render(m.statusMsg),
	)

	chatBox := styleBorder.Width(m.width - 2).Render(m.viewport.View())
	inputBar := styleBorder.Width(m.width - 2).Render(stylePrompt.Render("> ") + m.input.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		status,
		tabBar.String(),
		chatBox,
		inputBar,
	)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (m *Model) addLine(kind, text string) {
	tab := m.tabs[m.activeTab]
	m.messages[tab] = append(m.messages[tab], ChatLine{kind: kind, text: text})
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	tab := m.tabs[m.activeTab]
	var sb strings.Builder
	for _, l := range m.messages[tab] {
		sb.WriteString(l.text + "\n")
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m *Model) ensureTab(tab string) {
	if _, ok := m.messages[tab]; !ok {
		m.tabs = append(m.tabs, tab)
		m.messages[tab] = []ChatLine{}
	}
}

func indexOf(tabs []string, tab string) int {
	for i, t := range tabs {
		if t == tab {
			return i
		}
	}
	return 0
}

func removeTab(tabs []string, i int) []string {
	return append(tabs[:i], tabs[i+1:]...)
}
