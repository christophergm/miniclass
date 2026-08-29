import { fireEvent, screen, waitFor } from '@testing-library/react'
import { StrictMode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { renderWithQueryClient } from '@/test/queryClient'

import { GuardianRelationships } from './GuardianRelationships'
import { adultApi, guardianApi, studentApi, type Adult, type GuardianRelationship, type Student } from './roster'

afterEach(() => { vi.restoreAllMocks() })

const timestamps = { created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z' }
const ids = { organization_id: 'org-1', school_year_id: 'year-1' }

const morgan: Adult = { ...ids, ...timestamps, id: 'adult-1', legal_given_name: 'Morgan', legal_family_name: 'Lee', display_name: 'Morgan Lee', participation_intent: 'help' }
const riley: Student = { ...ids, ...timestamps, id: 'student-1', legal_given_name: 'Riley', legal_family_name: 'Stone', display_name: 'Riley Stone', grade_level_id: 'grade-3', homeroom_id: 'homeroom-a' }

function relationship(overrides: Partial<GuardianRelationship> = {}): GuardianRelationship {
  return { ...ids, ...timestamps, id: 'relationship-1', adult_id: 'adult-1', student_id: 'student-1', relationship_type: 'parent', ...overrides }
}

describe('guardian relationships', () => {
  it('edits a typed relationship from a student detail', async () => {
    vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([relationship()])
    vi.spyOn(adultApi, 'list').mockResolvedValue([morgan])
    const update = vi.spyOn(guardianApi, 'update').mockResolvedValue(relationship({ relationship_type: 'guardian' }))

    renderWithQueryClient(<MemoryRouter><GuardianRelationships kind="student" schoolYearId="year-1" personId="student-1" /></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Morgan Lee' })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Relationship for Morgan Lee'), { target: { value: 'guardian' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(update).toHaveBeenCalledWith('year-1', 'relationship-1', 'guardian'))
  })

  it('loads the reverse relationship view from an adult detail', async () => {
    vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([relationship({ relationship_type: 'grandparent' })])
    vi.spyOn(studentApi, 'list').mockResolvedValue([riley])

    renderWithQueryClient(<MemoryRouter><GuardianRelationships kind="adult" schoolYearId="year-1" personId="adult-1" /></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
    expect(screen.getByLabelText('Relationship for Riley Stone')).toHaveValue('grandparent')
  })

  // SPEC §15.3 and ADR 0013: an adult who is a guardian of nobody is a
  // legitimate volunteer record, so this is reported plainly and never as the
  // warning a student with no guardian gets.
  it('asks the API for only the selected adult’s relationships and does not warn when there are none', async () => {
    const listForAdult = vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([])
    vi.spyOn(studentApi, 'list').mockResolvedValue([riley])

    renderWithQueryClient(<MemoryRouter><GuardianRelationships kind="adult" schoolYearId="year-1" personId="adult-1" /></MemoryRouter>)

    expect(await screen.findByText('No guardian relationships recorded.')).toBeInTheDocument()
    expect(listForAdult).toHaveBeenCalledWith('year-1', 'adult-1')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(screen.queryByText(/no guardian yet/i)).not.toBeInTheDocument()
  })

  // SPEC §8.2 requires a warning and SPEC §5.2 forbids a block: nobody can be
  // reached about this child, and the roster record is still saveable. This is
  // the only warning of its kind on a person now (ADR 0012).
  it('warns that a student has no guardian without disabling anything', async () => {
    vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([])
    vi.spyOn(adultApi, 'list').mockResolvedValue([morgan])

    renderWithQueryClient(<MemoryRouter><GuardianRelationships kind="student" schoolYearId="year-1" personId="student-1" /></MemoryRouter>)

    const warning = await screen.findByText('This student has no guardian yet. Nobody can be reached about this child. This is a warning only; you can still save the roster record.')
    expect(warning).toHaveAttribute('role', 'status')
    expect(warning).toHaveClass('border-amber-300', 'bg-amber-50', 'text-amber-900')

    // The section still offers the fix rather than refusing the record.
    fireEvent.change(await screen.findByLabelText('Adult'), { target: { value: 'adult-1' } })
    expect(screen.getByRole('button', { name: 'Add relationship' })).toBeEnabled()
  })

  it('adds a relationship with the create endpoint and removes it by identifier', async () => {
    vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([relationship()])
    vi.spyOn(studentApi, 'list').mockResolvedValue([riley, { ...riley, id: 'student-2', display_name: 'Sam Stone' }])
    const create = vi.spyOn(guardianApi, 'create').mockResolvedValue(relationship({ id: 'relationship-2', student_id: 'student-2', relationship_type: 'other' }))
    const remove = vi.spyOn(guardianApi, 'remove').mockResolvedValue()

    renderWithQueryClient(<MemoryRouter><GuardianRelationships kind="adult" schoolYearId="year-1" personId="adult-1" /></MemoryRouter>)

    // Already-linked students are not offered again.
    const chooser = await screen.findByLabelText('Student')
    expect(screen.getByRole('option', { name: 'Sam Stone' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Riley Stone' })).not.toBeInTheDocument()

    fireEvent.change(chooser, { target: { value: 'student-2' } })
    fireEvent.change(screen.getByLabelText('New relationship type'), { target: { value: 'other' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add relationship' }))
    await waitFor(() => expect(create).toHaveBeenCalledWith('year-1', { adult_id: 'adult-1', student_id: 'student-2', relationship_type: 'other' }))

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(remove).toHaveBeenCalledWith('year-1', 'relationship-1'))
  })

  // StrictMode double-invokes effects in development, which the replaced
  // useEffect fetch answered with a second request to the network. React Query
  // dedupes the second mount onto the request already in flight.
  it('reads each list once per mount, including under StrictMode', async () => {
    const listForStudent = vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([relationship()])
    const list = vi.spyOn(adultApi, 'list').mockResolvedValue([morgan])

    renderWithQueryClient(<StrictMode><MemoryRouter><GuardianRelationships kind="student" schoolYearId="year-1" personId="student-1" /></MemoryRouter></StrictMode>)

    expect(await screen.findByRole('link', { name: 'Morgan Lee' })).toBeInTheDocument()
    expect(listForStudent).toHaveBeenCalledTimes(1)
    expect(list).toHaveBeenCalledTimes(1)
  })
})
