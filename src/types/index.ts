// -----------------------------------------------------------
// 🦉 A2A Network Types
// -----------------------------------------------------------

import { WebSocket } from 'ws';

export interface ExtendedWebSocket extends WebSocket {
    id: string;
    agentName: string;
    isAuthed: boolean;
    rateLimit: RateLimit;
    joinedRooms: Set<string>;
}

export interface Agent {
    id: string;
    name: string;
    socket: ExtendedWebSocket;
    isAuthed: boolean;
    rateLimit: RateLimit;
    joinedRooms: Set<string>;
}

export interface RateLimit {
    count: number;
    start: number;
}

export type MessageType = 'auth' | 'chat' | 'join' | 'leave' | 'list' | 'system' | 'welcome' | 'error' | 'ack';

export interface MessagePayload {
    type: MessageType;
    token?: string;       // Auth
    agentId?: string;     // Auth
    name?: string;        // Auth
    to?: string;          // Chat (Target ID or #Room)
    text?: string;        // Chat
    room?: string;        // Room ID
    from?: string;        // System Injected
    fromName?: string;    // System Injected
    scope?: 'public' | 'private' | 'room'; // System Injected
    online?: number;      // Welcome Stats
    agents?: { id: string, name: string }[]; // List Response
    rooms?: string[];     // List Response
    message?: string;     // Error/System Message
}
