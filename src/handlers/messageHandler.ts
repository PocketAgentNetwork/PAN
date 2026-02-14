// -----------------------------------------------------------
// 🦉 Message Handler
// -----------------------------------------------------------

import { WebSocket } from 'ws';
import { Agent, MessagePayload, ExtendedWebSocket } from '../types/index.js';
import { Config } from '../config/index.js';
import 'colors';

// ── State (In-Memory) ──
const agents = new Map<string, Agent>();
const rooms = new Map<string, Set<string>>();

export function handleConnection(ws: ExtendedWebSocket) {
    console.log(`[+] New connection: Wait for auth...`.yellow);
}

export function handleMessage(ws: ExtendedWebSocket, message: string) {
    let data: MessagePayload;

    // 0. Parse JSON
    try {
        data = JSON.parse(message);
    } catch (e) {
        ws.send(JSON.stringify({ type: "error", message: "Invalid JSON" }));
        return;
    }

    // 1. Rate Limiting
    const now = Date.now();
    if (now - ws.rateLimit.start > Config.RATE_LIMIT_WINDOW_MS) {
        ws.rateLimit = { count: 0, start: now };
    }
    ws.rateLimit.count++;

    if (ws.rateLimit.count > Config.MAX_MSGS_PER_WINDOW) {
        if (ws.rateLimit.count === Config.MAX_MSGS_PER_WINDOW + 1) {
            ws.send(JSON.stringify({ type: "error", message: "Rate limit exceeded." }));
            console.warn(`[!] Rate limit: ${ws.agentName || "Unknown"}`.yellow);
        }
        return;
    }

    // 2. Auth Check
    if (data.type === 'auth') {
        const { agentId, name, token } = data;

        if (token !== Config.SECRET_KEY) {
            console.log(`[!] Auth Failed: Invalid Token`.red);
            ws.send(JSON.stringify({ type: "error", message: "Invalid Token" }));
            ws.close();
            return;
        }

        if (!agentId || !name) {
            ws.send(JSON.stringify({ type: "error", message: "Missing agentId or name" }));
            return;
        }

        // Success
        ws.id = agentId;
        ws.agentName = name;
        ws.isAuthed = true;

        // Add to Registry
        agents.set(agentId, {
            id: agentId,
            name: name,
            socket: ws,
            isAuthed: true,
            rateLimit: ws.rateLimit,
            joinedRooms: ws.joinedRooms
        });

        console.log(`[✓] AUTH SUCCESS: ${name} (${agentId})`.green);

        ws.send(JSON.stringify({
            type: "welcome",
            message: `Welcome to A2A Network, ${name}. 🦉`,
            online: agents.size
        }));

        broadcast({ type: "system", message: `${name} joined.` }, ws.id);
        return;
    }

    // Require Auth
    if (!ws.isAuthed || !ws.id) {
        ws.send(JSON.stringify({ type: "error", message: "Auth required." }));
        return;
    }

    // 3. Routing Logic
    switch (data.type) {
        case 'chat':
            handleChat(ws, data);
            break;
        case 'join':
            handleRoomJoin(ws, data);
            break;
        case 'leave':
            handleRoomLeave(ws, data);
            break;
        case 'list':
            handleList(ws);
            break;
    }
}

function handleChat(sender: ExtendedWebSocket, data: MessagePayload) {
    const { to, text } = data;
    if (!text) return;

    // A. Broadcast
    if (to === 'all' || !to) {
        console.log(`[MSG] ${sender.agentName} -> ALL: ${text}`.cyan);
        broadcast({
            type: 'chat',
            from: sender.id,
            fromName: sender.agentName,
            text: text,
            scope: 'public'
        }, sender.id);
    }
    // B. Room
    else if (to.startsWith('#')) {
        const roomName = to;
        if (rooms.has(roomName) && rooms.get(roomName)!.has(sender.id)) {
            console.log(`[ROOM] ${sender.agentName} -> ${roomName}: ${text}`.blue);
            broadcastToRoom(roomName, {
                type: 'chat',
                from: sender.id,
                fromName: sender.agentName,
                to: roomName,
                text: text,
                scope: 'room'
            }, sender.id);
        } else {
            sender.send(JSON.stringify({ type: "error", message: `Not in room ${roomName}` }));
        }
    }
    // C. Direct Message
    else {
        const target = agents.get(to);
        if (target && target.socket.readyState === WebSocket.OPEN) {
            console.log(`[DM] ${sender.agentName} -> ${target.name}: ${text}`.magenta);
            target.socket.send(JSON.stringify({
                type: 'chat',
                from: sender.id,
                fromName: sender.agentName,
                text: text,
                scope: 'private'
            }));
            sender.send(JSON.stringify({ type: 'ack', message: `Sent to ${target.name}` }));
        } else {
            sender.send(JSON.stringify({ type: "error", message: "Agent not found" }));
        }
    }
}

function handleRoomJoin(ws: ExtendedWebSocket, data: MessagePayload) {
    const { room } = data;
    if (!room) return;

    if (!rooms.has(room)) rooms.set(room, new Set());

    rooms.get(room)!.add(ws.id);
    ws.joinedRooms.add(room);

    console.log(`[+] ${ws.agentName} joined ${room}`.blue);
    ws.send(JSON.stringify({ type: "system", message: `Joined ${room}` }));
}

function handleRoomLeave(ws: ExtendedWebSocket, data: MessagePayload) {
    const { room } = data;
    if (room && rooms.has(room)) {
        rooms.get(room)!.delete(ws.id);
        ws.joinedRooms.delete(room);
        ws.send(JSON.stringify({ type: "system", message: `Left ${room}` }));
    }
}

function handleList(ws: ExtendedWebSocket) {
    const online = Array.from(agents.values()).map(a => ({ id: a.id, name: a.name }));
    const activeRooms = Array.from(rooms.keys());
    ws.send(JSON.stringify({ type: 'list', agents: online, rooms: activeRooms }));
}

export function handleDisconnect(ws: ExtendedWebSocket) {
    if (ws.isAuthed && ws.id) {
        console.log(`[-] Disconnect: ${ws.agentName}`.red);
        agents.delete(ws.id);

        // Remove from rooms
        ws.joinedRooms.forEach((room: string) => {
            if (rooms.has(room)) rooms.get(room)!.delete(ws.id);
        });

        broadcast({ type: "system", message: `${ws.agentName} left.` });
    }
}

// ── Broadcast Helpers ──

function broadcast(msg: MessagePayload, excludeId?: string) {
    const payload = JSON.stringify(msg);
    agents.forEach(agent => {
        if (agent.id !== excludeId && agent.socket.readyState === WebSocket.OPEN) {
            agent.socket.send(payload);
        }
    });
}

function broadcastToRoom(roomName: string, msg: MessagePayload, excludeId?: string) {
    const payload = JSON.stringify(msg);
    const memberIds = rooms.get(roomName);
    if (!memberIds) return;

    memberIds.forEach(id => {
        if (id !== excludeId) {
            const agent = agents.get(id);
            if (agent && agent.socket.readyState === WebSocket.OPEN) {
                agent.socket.send(payload);
            }
        }
    });
}
