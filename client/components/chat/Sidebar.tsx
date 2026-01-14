'use client'

import { useState, useEffect } from 'react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Search, Plus, LogOut } from 'lucide-react'
import { useRouter } from 'next/navigation'

interface Contact {
  id: string
  uniqueId: string
  name: string
  email: string
  avatar?: string
  isOnline: boolean
  isContact: boolean
  lastMessage?: {
    content: string
    createdAt: string
  }
}

interface SidebarProps {
  currentUser: {
    id: string
    uniqueId: string
    email: string
    name: string
    avatar?: string
  }
  selectedContactId?: string
  onSelectContact: (contact: Contact) => void
  onAddContact: () => void
  refreshTrigger?: number
}

export default function Sidebar({
  currentUser,
  selectedContactId,
  onSelectContact,
  onAddContact,
  refreshTrigger,
}: SidebarProps) {
  const router = useRouter()
  const [contacts, setContacts] = useState<Contact[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    fetchContacts()
  }, [])

  useEffect(() => {
    if (refreshTrigger && refreshTrigger > 0) {
      fetchContacts()
    }
  }, [refreshTrigger])

  const fetchContacts = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/v1/messages/chats', {
        credentials: 'include',
      })

      if (response.ok) {
        const data = await response.json()
        // Map ChatListItem to Contact interface
        const mappedContacts: Contact[] = (data.data || []).map((item: {
          user: { id: string; uniqueId: string; name: string; email: string; avatar?: string };
          isOnline: boolean;
          isContact: boolean;
          lastMessage?: { content: string; createdAt: string };
        }) => ({
          id: item.user.id,
          uniqueId: item.user.uniqueId,
          name: item.user.name,
          email: item.user.email,
          avatar: item.user.avatar,
          isOnline: item.isOnline,
          isContact: item.isContact,
          lastMessage: item.lastMessage,
        }))
        // Deduplicate by ID while preserving order
        const seenIds = new Set<string>()
        const uniqueContacts: Contact[] = mappedContacts.filter((c) => {
          if (seenIds.has(c.id)) {
            return false
          }
          seenIds.add(c.id)
          return true
        })
        setContacts(uniqueContacts)
      }
    } catch (error) {
      console.error('Failed to fetch contacts:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleLogout = async () => {
    try {
      await fetch('http://localhost:8080/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      })
      router.push('/login')
    } catch (error) {
      console.error('Logout failed:', error)
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

  const filteredContacts = contacts.filter((contact) =>
    (contact.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    contact.uniqueId?.toLowerCase().includes(searchQuery.toLowerCase()))
  )

  return (
    <div className="flex h-screen w-80 flex-col border-r bg-background">
      {/* Header */}
      <div className="flex items-center justify-between border-b p-4">
        <div className="flex items-center gap-3">
          <Avatar className="h-10 w-10">
            <AvatarImage src={currentUser.avatar} alt={currentUser.name} />
            <AvatarFallback className="text-white" style={{ backgroundColor: '#0C2C55' }}>
              {getInitials(currentUser.name)}
            </AvatarFallback>
          </Avatar>
          <div className="flex flex-col">
            <span className="text-sm font-semibold">{currentUser.name}</span>
            <span className="text-xs text-muted-foreground">
              @{currentUser.uniqueId}
            </span>
          </div>
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={handleLogout}
          className="h-9 w-9"
        >
          <LogOut className="h-4 w-4" />
        </Button>
      </div>

      {/* Search Bar */}
      <div className="p-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search contacts..."
            value={searchQuery}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Add Contact Button */}
      <div className="px-4 pb-2">
        <Button
          onClick={onAddContact}
          className="w-full text-white hover:opacity-90"
          style={{ backgroundColor: '#0C2C55' }}
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Contact
        </Button>
      </div>

      <Separator />

      {/* Contact List */}
      <ScrollArea className="flex-1">
        {loading ? (
          <div className="space-y-2 p-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="flex items-center gap-3">
                <Skeleton className="h-12 w-12 rounded-full" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-3 w-24" />
                </div>
              </div>
            ))}
          </div>
        ) : filteredContacts.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-8 text-center">
            <div className="mb-4 rounded-full bg-muted p-4">
              <Search className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="mb-1 font-semibold">
              {searchQuery ? 'No contacts found' : 'No contacts yet'}
            </h3>
            <p className="mb-4 text-sm text-muted-foreground">
              {searchQuery
                ? 'Try a different search term'
                : 'Add your first contact to start chatting'}
            </p>
            {!searchQuery && (
              <Button onClick={onAddContact} size="sm" variant="outline">
                <Plus className="mr-2 h-4 w-4" />
                Add Contact
              </Button>
            )}
          </div>
        ) : (
          <div className="space-y-1 p-2">
            {filteredContacts.map((contact) => (
              <button
                key={contact.id}
                onClick={() => onSelectContact(contact)}
                className={`flex w-full items-center gap-3 rounded-lg p-3 text-left transition-colors hover:bg-muted/50 ${
                  selectedContactId === contact.id ? 'bg-muted' : ''
                }`}
              >
                <div className="relative">
                  <Avatar className="h-12 w-12">
                    <AvatarImage src={contact.avatar} alt={contact.name} />
                    <AvatarFallback className="bg-linear-to-br from-blue-500 to-cyan-500 text-white">
                      {getInitials(contact.name)}
                    </AvatarFallback>
                  </Avatar>
                  {contact.isOnline && (
                    <div className="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-background bg-green-500" />
                  )}
                </div>
                <div className="flex-1 overflow-hidden">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium truncate">{contact.name}</span>
                    <div className="flex items-center gap-1 shrink-0">
                      {contact.lastMessage && (
                        <span className="text-xs text-muted-foreground">
                          {new Date(contact.lastMessage.createdAt).toLocaleTimeString('en-US', {
                            hour: '2-digit',
                            minute: '2-digit',
                          })}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <p className="truncate text-sm text-muted-foreground">
                      {contact.lastMessage?.content || `@${contact.uniqueId}`}
                    </p>
                    <div className="flex items-center gap-1 shrink-0">
                      {!contact.isContact && (
                        <Badge
                          variant="secondary"
                          className="bg-orange-500/10 text-orange-600 text-xs"
                        >
                          Not in contacts
                        </Badge>
                      )}
                      {contact.isOnline && (
                        <Badge
                          variant="secondary"
                          className="bg-green-500/10 text-green-600 text-xs"
                        >
                          Online
                        </Badge>
                      )}
                    </div>
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  )
}
