import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { ModalForm } from './modal-form'

describe('ModalForm', () => {
  it('focuses the form and restores focus after Escape', () => {
    const onClose = vi.fn()
    function Harness() {
      const [open, setOpen] = useState(false)
      return <><button onClick={() => setOpen(true)} type="button">Open</button><ModalForm onClose={() => { onClose(); setOpen(false) }} open={open} title="Edit session"><input aria-label="Session name" /></ModalForm></>
    }
    render(<Harness />)
    const opener = screen.getByRole('button', { name: 'Open' })
    opener.focus()
    fireEvent.click(opener)
    expect(screen.getByLabelText('Session name')).toHaveFocus()

    fireEvent.keyDown(document, { key: 'Escape' })

    expect(onClose).toHaveBeenCalledOnce()
    expect(opener).toHaveFocus()
  })

  it('guards dirty dismissal and keeps the dialog open when discard is declined', () => {
    const onClose = vi.fn()
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<ModalForm dirty onClose={onClose} open title="Edit session"><input aria-label="Session name" /></ModalForm>)

    fireEvent.click(screen.getByRole('button', { name: 'Close dialog' }))

    expect(window.confirm).toHaveBeenCalledWith('Discard unsaved changes?')
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog', { name: 'Edit session' })).toBeInTheDocument()
    vi.restoreAllMocks()
  })

  it('closes dirty forms when discard is confirmed', () => {
    const onClose = vi.fn()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    render(<ModalForm dirty onClose={onClose} open title="Edit session"><input aria-label="Session name" /></ModalForm>)

    fireEvent.keyDown(document, { key: 'Escape' })

    expect(onClose).toHaveBeenCalledOnce()
    vi.restoreAllMocks()
  })
})
