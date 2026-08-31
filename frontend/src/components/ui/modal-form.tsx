import { useEffect, useRef, type ReactNode } from 'react'

type ModalFormProps = {
  open: boolean
  title: string
  description?: string
  dirty?: boolean
  onClose: () => void
  children: ReactNode
}

function focusableElements(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])'))
}

/** A small, dependency-free modal shell for compact authoring forms. */
export function ModalForm({ open, title, description, dirty = false, onClose, children }: ModalFormProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const openerRef = useRef<HTMLElement | null>(null)
  const closeRef = useRef(onClose)
  const dirtyRef = useRef(dirty)

  useEffect(() => {
    closeRef.current = onClose
    dirtyRef.current = dirty
  }, [dirty, onClose])

  useEffect(() => {
    if (!open) return
    openerRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const dialog = dialogRef.current
    const focusable = dialog && focusableElements(dialog)
    const first = focusable && (focusable[1] ?? focusable[0])
    first?.focus()

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        event.preventDefault()
        requestClose()
        return
      }
      if (event.key !== 'Tab' || !dialog) return
      const focusable = focusableElements(dialog)
      if (focusable.length === 0) {
        event.preventDefault()
        return
      }
      const firstFocusable = focusable[0]
      const lastFocusable = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === firstFocusable) {
        event.preventDefault()
        lastFocusable.focus()
      } else if (!event.shiftKey && document.activeElement === lastFocusable) {
        event.preventDefault()
        firstFocusable.focus()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      openerRef.current?.focus()
    }

    function requestClose() {
      if (!dirtyRef.current || window.confirm('Discard unsaved changes?')) closeRef.current()
    }
  }, [open])

  if (!open) return null

  function requestClose() {
    if (!dirty || window.confirm('Discard unsaved changes?')) onClose()
  }

  return <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/50 px-4 py-10" onMouseDown={(event) => { if (event.target === event.currentTarget) requestClose() }}>
    <div aria-describedby={description ? 'modal-form-description' : undefined} aria-labelledby="modal-form-title" aria-modal="true" className="w-full max-w-lg rounded-lg border bg-card p-6 shadow-xl" ref={dialogRef} role="dialog">
      <div className="flex items-start justify-between gap-4"><div><h2 className="text-lg font-semibold" id="modal-form-title">{title}</h2>{description && <p className="mt-1 text-sm text-muted-foreground" id="modal-form-description">{description}</p>}</div><button aria-label="Close dialog" className="rounded-md px-2 py-1 text-lg leading-none text-muted-foreground hover:bg-accent hover:text-foreground" onClick={requestClose} type="button">×</button></div>
      <div className="mt-5">{children}</div>
    </div>
  </div>
}
