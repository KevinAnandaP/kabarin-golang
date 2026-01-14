'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'
import Sidebar from '@/components/chat/Sidebar'
import ChatArea from '@/components/chat/ChatArea'
import AddContactModal from '@/components/chat/AddContactModal'

interface User {
  id: string
  uniqueId: string
  email: string
  name: string
  avatar?: string
  isOnline: boolean
}

interface Contact {
  id: string
  uniqueId: string
  name: string
  email: string
  avatar?: string
  isOnline: boolean
}

export default function ChatPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedContact, setSelectedContact] = useState<Contact>()
  const [showAddContact, setShowAddContact] = useState(false)
  const [refreshContacts, setRefreshContacts] = useState(0)

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const response = await fetch('http://localhost:8080/api/v1/auth/me', {
          credentials: 'include',
        })

        if (!response.ok) {
          router.push('/login')
          return
        }

        const data = await response.json()
        setUser(data.data)
      } catch {
        router.push('/login')
      } finally {
        setLoading(false)
      }
    }

    checkAuth()
  }, [router])

  const handleContactAdded = () => {
    setRefreshContacts((prev) => prev + 1)
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!user) {
    return null
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar
        currentUser={user}
        selectedContactId={selectedContact?.id}
        onSelectContact={setSelectedContact}
        onAddContact={() => setShowAddContact(true)}
        refreshTrigger={refreshContacts}
      />
      <ChatArea 
        contact={selectedContact} 
        currentUserId={user.id}
        onContactAdded={handleContactAdded}
      />
      <AddContactModal
        open={showAddContact}
        onClose={() => setShowAddContact(false)}
        onContactAdded={handleContactAdded}
      />
    </div>
  )
}
