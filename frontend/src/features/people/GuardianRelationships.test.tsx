import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { guardianApi } from './guardianApi'
import { GuardianRelationships } from './GuardianRelationships'
import { peopleApi } from './api'

afterEach(() => { vi.restoreAllMocks() })

const morgan = { id: 'adult-1', school_year_id: 'year-1', legal_given_name: 'Morgan', legal_family_name: 'Lee', display_name: 'Morgan Lee', participation_intent: 'help' as const }
const riley = { id: 'student-1', school_year_id: 'year-1', legal_given_name: 'Riley', legal_family_name: 'Stone', display_name: 'Riley Stone', grade: '3', homeroom: 'A' }

describe('guardian relationships', () => {
  it('edits a typed relationship from a student detail without changing membership', async () => {
    vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([{
      id: 'relationship-1', school_year_id: 'year-1', adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'parent',
    }])
    vi.spyOn(peopleApi, 'list').mockResolvedValue([morgan])
    const update = vi.spyOn(guardianApi, 'update').mockResolvedValue({ id: 'relationship-1', school_year_id: 'year-1', adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'guardian' })

    render(<MemoryRouter><GuardianRelationships kind="student" schoolYearId="year-1" personId="student-1" /></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Morgan Lee' })).toBeInTheDocument()
    expect(screen.getByText(/They are separate from household membership/)).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Relationship for Morgan Lee'), { target: { value: 'guardian' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(update).toHaveBeenCalledWith('year-1', 'relationship-1', 'guardian'))
  })

  it('loads the reverse relationship view from an adult detail', async () => {
    vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([{
      id: 'relationship-1', school_year_id: 'year-1', adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'grandparent',
    }])
    vi.spyOn(peopleApi, 'list').mockResolvedValue([riley])

    render(<MemoryRouter><GuardianRelationships kind="adult" schoolYearId="year-1" personId="adult-1" /></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
    expect(screen.getByLabelText('Relationship for Riley Stone')).toHaveValue('grandparent')
  })

  it('asks the API for only the selected adult’s relationships', async () => {
    const listForAdult = vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([])
    vi.spyOn(peopleApi, 'list').mockResolvedValue([riley])

    render(<MemoryRouter><GuardianRelationships kind="adult" schoolYearId="year-1" personId="adult-1" /></MemoryRouter>)

    expect(await screen.findByText('No guardian relationships recorded.')).toBeInTheDocument()
    expect(listForAdult).toHaveBeenCalledWith('year-1', 'adult-1')
  })

  it('adds a relationship with the create endpoint and removes it by identifier', async () => {
    vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([{
      id: 'relationship-1', school_year_id: 'year-1', adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'parent',
    }])
    vi.spyOn(peopleApi, 'list').mockResolvedValue([riley, { ...riley, id: 'student-2', display_name: 'Sam Stone' }])
    const create = vi.spyOn(guardianApi, 'create').mockResolvedValue({ id: 'relationship-2', school_year_id: 'year-1', adult_id: 'adult-1', student_id: 'student-2', relationship_type: 'other' })
    const remove = vi.spyOn(guardianApi, 'remove').mockResolvedValue()

    render(<MemoryRouter><GuardianRelationships kind="adult" schoolYearId="year-1" personId="adult-1" /></MemoryRouter>)

    // Already-linked students are not offered again.
    const chooser = await screen.findByLabelText('Student')
    expect(screen.getByRole('option', { name: 'Sam Stone' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Riley Stone' })).not.toBeInTheDocument()

    fireEvent.change(chooser, { target: { value: 'student-2' } })
    fireEvent.change(screen.getByLabelText('New relationship type'), { target: { value: 'other' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add relationship' }))
    await waitFor(() => expect(create).toHaveBeenCalledWith('year-1', 'adult-1', 'student-2', 'other'))

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(remove).toHaveBeenCalledWith('year-1', 'relationship-1'))
  })
})
