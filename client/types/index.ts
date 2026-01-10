// Base types untuk aplikasi Ngabarin

export interface User {
  id: string;
  uniqueId: string; // Format: #GOPRO-882
  email: string;
  name: string;
  avatar?: string;
  authProvider: 'email' | 'google';
  createdAt: string;
  updatedAt: string;
}

export interface Contact {
  id: string;
  userId: string;
  contactId: string;
  contact: User;
  addedAt: string;
  isOnline: boolean;
}

export interface Message {
  id: string;
  senderId: string;
  receiverId?: string; // untuk 1-on-1
  groupId?: string; // untuk group chat
  content: string;
  type: 'text' | 'image' | 'file';
  status: 'sent' | 'delivered' | 'read';
  createdAt: string;
  updatedAt: string;
}

export interface Group {
  id: string;
  name: string;
  icon?: string;
  createdBy: string;
  members: GroupMember[];
  createdAt: string;
  updatedAt: string;
}

export interface GroupMember {
  groupId: string;
  userId: string;
  user: User;
  joinedAt: string;
}

export interface ChatRoom {
  id: string;
  type: 'direct' | 'group';
  lastMessage?: Message;
  unreadCount: number;
  participant?: User; // untuk direct chat
  group?: Group; // untuk group chat
}

// WebSocket event types
export interface WSMessage {
  type: 'message' | 'typing' | 'status' | 'online' | 'offline';
  payload: Message | TypingIndicator | { userId: string; status: string } | unknown;
}

export interface TypingIndicator {
  userId: string;
  chatId: string;
  isTyping: boolean;
}
