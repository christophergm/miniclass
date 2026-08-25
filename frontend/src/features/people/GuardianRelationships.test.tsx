import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { guardianApi } from './guardianApi'
import { GuardianRelationships } from './GuardianRelationships'
import { peopleApi } from './api'

afterEach(() => { vi.restoreAllMocks() })

describe('guardian relationships', () => {
  it('edits a typed relationship from a student detail without changing membership', async () => {
    vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([{
      adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'parent', adult: { id: 'adult-1', school_year_id: 'year-1', legal_given_name: 'Morgan', legal_family_name: 'Lee', display_name: 'Morgan Lee', participation_intent: 'help' },
    }])
    vi.spyOn(peopleApi, 'list').mockResolvedValue([])
    const save = vi.spyOn(guardianApi, 'save').mockResolvedValue({ adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'guardian' })

    render(<MemoryRouter><GuardianRelationships kind="student" schoolYearId="year-1" personId="student-1" /></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Morgan Lee' })).toBeInTheDocument()
    expect(screen.getByText(/They are separate from household membership/)).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Relationship for Morgan Lee'), { target: { value: 'guardian' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(save).toHaveBeenCalledWith('year-1', 'adult-1', 'student-1', 'guardian'))
  })

  it('loads the reverse relationship view from an adult detail', async () => {
    vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([{
      adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'grandparent', student: { id: 'student-1', school_year_id: 'year-1', legal_given_name: 'Riley', legal_family_name: 'Stone', display_name: 'Riley Stone', grade: '3', homeroom: 'A' },
    }])
    vi.spyOn(peopleApi, 'list').mockResolvedValue([])

    render(<MemoryRouter><GuardianRelationships kind="adult" schoolYearId="year-1" personId="adult-1" /></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
    expect(screen.getByLabelText('Relationship for Riley Stone')).toHaveValue('grandparent')
  })
})
