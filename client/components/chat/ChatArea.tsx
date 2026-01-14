'use client'

import { useState, useEffect, useRef } from 'react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Send, Paperclip, MessageSquare, UserPlus, Loader2, Check, CheckCheck } from 'lucide-react'

interface Contact {
  id: string
  uniqueId: string
  name: string
  email: string
  avatar?: string
  isOnline: boolean
  isContact?: boolean
}

interface Message {
  id: string
  senderId: string
  receiverId: string
  content: string
  status: string
  createdAt: string
}

interface MessageWithSender {
  id: string
  senderId?: string
  receiverId: string
  content: string
  status: string
  createdAt: string
  sender?: {
    id: string
  }
}

interface ChatAreaProps {
  contact?: Contact
  currentUserId: string
  onContactAdded?: () => void
}

export default function ChatArea({ contact, currentUserId, onContactAdded }: ChatAreaProps) {
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')
  const [sending, setSending] = useState(false)
  const [addingContact, setAddingContact] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (contact) {
      fetchMessages()
    }
  }, [contact])

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const fetchMessages = async () => {
    if (!contact) return

    setLoading(true)
    try {
      const response = await fetch(
        `http://localhost:8080/api/v1/messages/${contact.id}`,
        {
          credentials: 'include',
        }
      )

      if (response.ok) {
        const data = await response.json()
        // Backend returns { data: { messages: [...] } }
        const messageList = data.data?.messages || []
        // Map MessageWithSender to Message interface
        const mappedMessages = messageList.map((msg: MessageWithSender) => ({
          id: msg.id,
          senderId: msg.sender?.id || msg.senderId,
          receiverId: msg.receiverId,
          content: msg.content,
          status: msg.status,
          createdAt: msg.createdAt,
        }))
        setMessages(mappedMessages)
      }
    } catch (error) {
      console.error('Failed to fetch messages:', error)
    } finally {
      setLoading(false)
    }
  }

  const sendMessage = async () => {
    if (!message.trim() || !contact || sending) return

    setSending(true)
    try {
      const response = await fetch('http://localhost:8080/api/v1/messages', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          receiverId: contact.id,
          content: message.trim(),
        }),
      })

      if (response.ok) {
        const data = await response.json()
        setMessages([...messages, data.data])
        setMessage('')
      }
    } catch (error) {
      console.error('Failed to send message:', error)
    } finally {
      setSending(false)
    }
  }

  const scrollToBottom = () => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }

  const handleAddContact = async () => {
    if (!contact || addingContact) return

    setAddingContact(true)
    try {
      const response = await fetch('http://localhost:8080/api/v1/contacts', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          uniqueId: contact.uniqueId,
        }),
      })

      if (response.ok) {
        // Update contact status locally
        if (contact) {
          contact.isContact = true
        }
        // Trigger refresh in parent
        if (onContactAdded) {
          onContactAdded()
        }
      }
    } catch (error) {
      console.error('Failed to add contact:', error)
    } finally {
      setAddingContact(false)
    }
  }

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }

  const formatTime = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  if (!contact) {
    return (
      <div className="flex h-screen flex-1 flex-col items-center justify-center bg-muted/30">
        <div className="rounded-full bg-muted p-6">
          <MessageSquare className="h-12 w-12 text-muted-foreground" />
        </div>
        <h3 className="mt-4 text-xl font-semibold">Select a chat</h3>
        <p className="text-sm text-muted-foreground">
          Choose a contact to start messaging
        </p>
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-1 flex-col">
      {/* Non-contact Alert */}
      {contact.isContact === false && (
        <Alert className="m-4 border-orange-500 bg-orange-50">
          <AlertDescription className="flex items-center justify-between">
            <span className="text-sm text-orange-900">
              This person is not in your contacts
            </span>
            <Button
              size="sm"
              onClick={handleAddContact}
              disabled={addingContact}
              className="text-white hover:opacity-90"
              style={{ backgroundColor: '#0C2C55' }}
            >
              {addingContact ? (
                <>
                  <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                  Adding...
                </>
              ) : (
                <>
                  <UserPlus className="mr-2 h-3 w-3" />
                  Add to Contacts
                </>
              )}
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Chat Header */}
      <div className="flex items-center gap-3 border-b p-4">
        <div className="relative">
          <Avatar className="h-10 w-10">
            <AvatarImage src={contact.avatar} alt={contact.name} />
            <AvatarFallback className="bg-linear-to-br from-blue-500 to-cyan-500 text-white">
              {getInitials(contact.name)}
            </AvatarFallback>
          </Avatar>
          {contact.isOnline && (
            <div className="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-background bg-green-500" />
          )}
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h2 className="font-semibold">{contact.name}</h2>
            {contact.isOnline && (
              <Badge
                variant="secondary"
                className="bg-green-500/10 text-green-600"
              >
                Online
              </Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground">@{contact.uniqueId}</p>
        </div>
      </div>

      {/* Messages Area */}
      <ScrollArea className="flex-1 p-4" ref={scrollRef}>
        {loading ? (
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <div
                key={i}
                className={`flex ${i % 2 === 0 ? 'justify-start' : 'justify-end'}`}
              >
                <div className="flex max-w-[70%] gap-2">
                  {i % 2 === 0 && <Skeleton className="h-8 w-8 rounded-full" />}
                  <Skeleton className="h-16 w-48 rounded-2xl" />
                  {i % 2 !== 0 && <Skeleton className="h-8 w-8 rounded-full" />}
                </div>
              </div>
            ))}
          </div>
        ) : messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center text-center">
            <div className="mb-4 rounded-full bg-muted p-4">
              <MessageSquare className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="mb-1 font-semibold">No messages yet</h3>
            <p className="text-sm text-muted-foreground">
              Send a message to start the conversation
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {messages.map((msg) => {
              const isOwnMessage = msg.senderId === currentUserId
              return (
                <div
                  key={msg.id}
                  className={`flex ${isOwnMessage ? 'justify-end' : 'justify-start'}`}
                >
                  <div
                    className={`flex max-w-[70%] gap-2 ${isOwnMessage ? 'flex-row-reverse' : 'flex-row'}`}
                  >
                    <Avatar className="h-8 w-8">
                      <AvatarImage
                        src={isOwnMessage ? undefined : contact.avatar}
                        alt={isOwnMessage ? 'You' : contact.name}
                      />
                      <AvatarFallback
                        className="text-xs text-white"
                        style={{
                          backgroundColor: isOwnMessage ? '#0C2C55' : '#3b82f6'
                        }}
                      >
                        {getInitials(isOwnMessage ? 'You' : contact.name)}
                      </AvatarFallback>
                    </Avatar>
                    <div
                      className={`flex flex-col ${isOwnMessage ? 'items-end' : 'items-start'}`}
                    >
                      <div
                        className={`rounded-2xl px-4 py-2 ${
                          isOwnMessage
                            ? 'text-white'
                            : 'bg-muted'
                        }`}
                        style={isOwnMessage ? { backgroundColor: '#0C2C55' } : undefined}
                      >
                        <p className="text-sm">{msg.content}</p>
                      </div>
                      <div className="mt-1 flex items-center gap-1">
                        <span className="text-xs text-muted-foreground">
                          {formatTime(msg.createdAt)}
                        </span>
                        {isOwnMessage && (
                          <span className="flex items-center">
                            {msg.status === 'read' ? (
                              <CheckCheck className="h-3 w-3 text-blue-500" />
                            ) : msg.status === 'delivered' ? (
                              <CheckCheck className="h-3 w-3 text-muted-foreground" />
                            ) : (
                              <Check className="h-3 w-3 text-muted-foreground" />
                            )}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </ScrollArea>

      {/* Message Input */}
      <div className="border-t p-4">
        <div className="flex gap-2">
          <Button variant="outline" size="icon" className="shrink-0">
            <Paperclip className="h-4 w-4" />
          </Button>
          <Textarea
            placeholder="Type a message..."
            value={message}
            onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
              setMessage(e.target.value)
            }
            onKeyDown={handleKeyDown}
            className="min-h-11 max-h-32 resize-none"
            rows={1}
          />
          <Button
            onClick={sendMessage}
            disabled={!message.trim() || sending}
            className="shrink-0 text-white hover:opacity-90"
            style={{ backgroundColor: '#0C2C55' }}
            size="icon"
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
