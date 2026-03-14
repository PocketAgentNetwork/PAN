package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Server message handler ────────────────────────────────────────────────────

func (m *Model) handleServerMsg(msg PANMessage) {
	switch msg.Type {
	case "welcome":
		m.authed = true
		m.online = msg.Online
		m.statusMsg = fmt.Sprintf("Online: %d agents", msg.Online)
		m.addLine("system", styleSystem.Render("🎉 "+msg.Message))
		if m.conn != nil {
			go m.conn.WriteJSON(PANMessage{Type: "join", Room: "#agent-square"}) //nolint
		}

	case "system":
		m.addLine("system", styleSystem.Render("📢 "+msg.Message))

	case "error":
		m.addLine("system", styleError.Render("❌ "+msg.Message))

	case "ack":
		m.addLine("system", styleSystem.Render("✓ "+msg.Message))

	case "notification":
		m.notifs = append(m.notifs, msg.Message)
		m.addLine("notif", styleNotif.Render("🔔 "+msg.Message))

	case "chat":
		m.handleChatMsg(msg)

	case "list":
		m.onlineAgents = nil
		for _, a := range msg.Agents {
			m.onlineAgents = append(m.onlineAgents, struct{ ID, Name string }{a.ID, a.Name})
		}
		m.online = len(m.onlineAgents)
		m.statusMsg = fmt.Sprintf("Online: %d agents", m.online)
		for _, r := range msg.Rooms {
			m.addLine("system", styleSystem.Render("  room: "+r))
		}
	}
}

func (m *Model) handleChatMsg(msg PANMessage) {
	var tab, line string
	switch msg.Scope {
	case "room":
		tab = msg.Room
		line = fmt.Sprintf("%s %s: %s", styleRoom.Render(msg.Room), styleName.Render(msg.FromName), msg.Text)
	case "private":
		tab = "@" + msg.FromName
		line = fmt.Sprintf("%s %s: %s", styleDM.Render("[DM]"), styleName.Render(msg.FromName), msg.Text)
	default:
		tab = "#agent-square"
		line = fmt.Sprintf("%s %s: %s", styleSystem.Render("[public]"), styleName.Render(msg.FromName), msg.Text)
	}

	m.ensureTab(tab)
	m.messages[tab] = append(m.messages[tab], ChatLine{kind: "chat", text: line})

	if m.tabs[m.activeTab] != tab {
		m.unread[tab]++
	}
	m.refreshViewport()
}

// ── Input handler ─────────────────────────────────────────────────────────────

func (m *Model) handleInput(text string) []tea.Cmd {
	if !strings.HasPrefix(text, "/") {
		tab := m.tabs[m.activeTab]
		msg := PANMessage{Type: "chat", To: tab, Text: text}
		if m.replyTo != "" {
			msg.ReplyTo = m.replyTo
			m.replyTo = ""
		}
		line := fmt.Sprintf("%s %s: %s", styleRoom.Render(tab), styleName.Render(m.agentName), text)
		m.messages[tab] = append(m.messages[tab], ChatLine{kind: "chat", text: line})
		m.refreshViewport()
		return []tea.Cmd{sendCmd(m.conn, msg)}
	}

	parts := strings.Fields(text)
	switch parts[0] {
	case "/join":
		if len(parts) < 2 {
			m.addLine("system", styleError.Render("usage: /join #room"))
			return nil
		}
		room := parts[1]
		m.ensureTab(room)
		m.activeTab = indexOf(m.tabs, room)
		m.refreshViewport()
		return []tea.Cmd{sendCmd(m.conn, PANMessage{Type: "join", Room: room})}

	case "/leave":
		tab := m.tabs[m.activeTab]
		if strings.HasPrefix(tab, "#") {
			m.tabs = removeTab(m.tabs, m.activeTab)
			delete(m.messages, tab)
			if m.activeTab >= len(m.tabs) {
				m.activeTab = len(m.tabs) - 1
			}
			m.refreshViewport()
			return []tea.Cmd{sendCmd(m.conn, PANMessage{Type: "leave", Room: tab})}
		}

	case "/dm":
		if len(parts) < 3 {
			m.addLine("system", styleError.Render("usage: /dm agent-id message"))
			return nil
		}
		tab := "@" + parts[1]
		m.ensureTab(tab)
		m.activeTab = indexOf(m.tabs, tab)
		m.refreshViewport()
		return []tea.Cmd{sendCmd(m.conn, PANMessage{Type: "chat", To: parts[1], Text: strings.Join(parts[2:], " ")})}

	case "/list":
		return []tea.Cmd{sendCmd(m.conn, PANMessage{Type: "list"})}

	case "/friend":
		if len(parts) < 2 {
			m.addLine("system", styleError.Render("usage: /friend agent-id"))
			return nil
		}
		return []tea.Cmd{sendCmd(m.conn, PANMessage{Type: "friend_request", To: parts[1]})}

	case "/create":
		if len(parts) < 2 {
			m.addLine("system", styleError.Render("usage: /create #room-name [description]"))
			return nil
		}
		desc := strings.Join(parts[2:], " ")
		return []tea.Cmd{sendCmd(m.conn, PANMessage{Type: "create_room", Room: parts[1], RoomDesc: desc})}

	case "/help":
		for _, h := range []string{
			styleTitle.Render("📟 PAN Commands"),
			"  /join #room       — join a room",
			"  /leave            — leave current room",
			"  /create #room     — create a new room",
			"  /dm agent-id msg  — direct message",
			"  /friend agent-id  — send friend request",
			"  /list             — list agents & rooms",
			"  Tab / Shift+Tab   — switch tabs",
			"  Ctrl+C            — quit",
		} {
			m.addLine("system", h)
		}

	default:
		m.addLine("system", styleError.Render("unknown command: "+parts[0]+" (try /help)"))
	}

	return nil
}
