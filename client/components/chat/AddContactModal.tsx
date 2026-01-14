'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, UserPlus } from 'lucide-react'

interface AddContactModalProps {
  open: boolean
  onClose: () => void
  onContactAdded: () => void
}

export default function AddContactModal({
  open,
  onClose,
  onContactAdded,
}: AddContactModalProps) {
  const [uniqueId, setUniqueId] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!uniqueId.trim()) return

    setLoading(true)
    setError('')
    setSuccess(false)

    try {
      const response = await fetch('http://localhost:8080/api/v1/contacts', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          uniqueId: uniqueId.trim(),
        }),
      })

      const data = await response.json()

      if (response.ok) {
        setSuccess(true)
        setUniqueId('')
        setTimeout(() => {
          onContactAdded()
          onClose()
          setSuccess(false)
        }, 1000)
      } else {
        setError(data.error || 'Failed to add contact')
      }
    } catch {
      setError('Failed to add contact. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    if (!loading) {
      setUniqueId('')
      setError('')
      setSuccess(false)
      onClose()
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UserPlus className="h-5 w-5" />
            Add New Contact
          </DialogTitle>
          <DialogDescription>
            Enter the unique ID of the person you want to add as a contact.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="uniqueId">Unique ID</Label>
            <Input
              id="uniqueId"
              placeholder="e.g., john_doe_1234"
              value={uniqueId}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setUniqueId(e.target.value)
              }
              disabled={loading}
              autoFocus
            />
            <p className="text-xs text-muted-foreground">
              The unique ID appears below the user&apos;s name with an @ symbol
            </p>
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {success && (
            <Alert className="border-green-500 bg-green-50 text-green-900">
              <AlertDescription>Contact added successfully!</AlertDescription>
            </Alert>
          )}

          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={loading}
              className="flex-1"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!uniqueId.trim() || loading}
              className="flex-1 text-white hover:opacity-90"
              style={{ backgroundColor: '#0C2C55' }}
            >
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Adding...
                </>
              ) : (
                <>
                  <UserPlus className="mr-2 h-4 w-4" />
                  Add Contact
                </>
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
