import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { HouseholdDetailPage, HouseholdListPage } from './HouseholdPages'
import { adultApi, householdApi, studentApi, type Household, type Student } from './roster'

const timestamps = { created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z' }
const ids = { organization_id: 'org-1', school_year_id: 'year-1' }

const student: Student = { ...ids, ...timestamps, id: 'student-1', legal_given_name: 'Riley', legal_family_name: 'Stone', display_name: 'Riley Stone', grade_level_id: 'grade-3', homeroom_id: 'homeroom-a' }
const otherStudent: Student = { ...student, id: 'student-2', legal_given_name: 'Jordan', legal_family_name: 'Reed', display_name: 'Jordan Reed' }

const householdOne: Household = { ...ids, ...timestamps, id: 'household-1', display_name: 'Stone family' }
const householdTwo: Household = { ...ids, ...timestamps, id: 'household-2', display_name: 'Stone second home' }

// Membership rows carry identifiers only; the display name has to be joined
// from the roster (SPEC §8.7), which is what these fixtures exercise.
function membership(householdId: string, studentId: string) {
  return { id: `${householdId}-${studentId}`, household_id: householdId, student_id: studentId }
}

afterEach(() => { vi.restoreAllMocks() })

describe('household pages', () => {
  it('lists households with member counts and keeps students grouped separately from adults', async () => {
    vi.spyOn(householdApi, 'list').mockResolvedValue([householdOne, householdTwo])
    vi.spyOn(householdApi, 'listStudents').mockImplementation(async (_year, householdId) => [membership(householdId, 'student-1')])
    vi.spyOn(householdApi, 'listAdults').mockResolvedValue([])

    render(<MemoryRouter initialEntries={['/y/year-1/households']}><Routes><Route path="/y/:schoolYearId/households" element={<HouseholdListPage />} /></Routes></MemoryRouter>)

    const table = await screen.findByRole('table', { name: 'Households' })
    expect(within(table).getByRole('link', { name: 'Stone family' })).toBeInTheDocument()
    expect(within(table).getByRole('link', { name: 'Stone second home' })).toBeInTheDocument()
    expect(within(table).getByText('Students')).toBeInTheDocument()
    expect(within(table).getByText('Adults')).toBeInTheDocument()

    const stoneRow = within(table).getByRole('link', { name: 'Stone family' }).closest('tr')!
    expect(within(stoneRow).getAllByRole('cell')[1]).toHaveTextContent('1')
    expect(within(stoneRow).getAllByRole('cell')[2]).toHaveTextContent('0')
  })

  it('renders a student in a household detail and manages membership independently', async () => {
    vi.spyOn(householdApi, 'get').mockResolvedValue(householdOne)
    vi.spyOn(householdApi, 'listStudents').mockResolvedValue([membership('household-1', 'student-1')])
    vi.spyOn(householdApi, 'listAdults').mockResolvedValue([])
    vi.spyOn(studentApi, 'list').mockResolvedValue([student, otherStudent])
    vi.spyOn(adultApi, 'list').mockResolvedValue([])
    const addStudent = vi.spyOn(householdApi, 'addStudent').mockResolvedValue({ id: 'membership-2', household_id: 'household-1', student_id: 'student-2' })
    const removeStudent = vi.spyOn(householdApi, 'removeStudent').mockResolvedValue()

    render(<MemoryRouter initialEntries={['/y/year-1/households/household-1']}><Routes><Route path="/y/:schoolYearId/households/:householdId" element={<HouseholdDetailPage />} /></Routes></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
    expect(screen.getByText(/Membership controls who is grouped with this household/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(removeStudent).toHaveBeenCalledWith('year-1', 'household-1', 'student-1'))

    fireEvent.change(screen.getByLabelText('Person'), { target: { value: 'student-2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add member' }))
    await waitFor(() => expect(addStudent).toHaveBeenCalledWith('year-1', 'household-1', 'student-2'))
  })

  it('offers only the people who are not already members', async () => {
    vi.spyOn(householdApi, 'get').mockResolvedValue(householdOne)
    vi.spyOn(householdApi, 'listStudents').mockResolvedValue([membership('household-1', 'student-1')])
    vi.spyOn(householdApi, 'listAdults').mockResolvedValue([])
    vi.spyOn(studentApi, 'list').mockResolvedValue([student, otherStudent])
    vi.spyOn(adultApi, 'list').mockResolvedValue([])

    render(<MemoryRouter initialEntries={['/y/year-1/households/household-1']}><Routes><Route path="/y/:schoolYearId/households/:householdId" element={<HouseholdDetailPage />} /></Routes></MemoryRouter>)

    const picker = await screen.findByLabelText('Person')
    await waitFor(() => expect(picker).toContainHTML('Jordan Reed'))
    expect(picker).not.toContainHTML('Riley Stone')
  })

  it('renders a student belonging to two households in both household details', async () => {
    vi.spyOn(householdApi, 'get').mockImplementation(async (_schoolYearId, householdId) => householdId === 'household-1' ? householdOne : householdTwo)
    vi.spyOn(householdApi, 'listStudents').mockImplementation(async (_year, householdId) => [membership(householdId, 'student-1')])
    vi.spyOn(householdApi, 'listAdults').mockResolvedValue([])
    vi.spyOn(studentApi, 'list').mockResolvedValue([student])
    vi.spyOn(adultApi, 'list').mockResolvedValue([])

    const first = render(<MemoryRouter initialEntries={['/y/year-1/households/household-1']}><Routes><Route path="/y/:schoolYearId/households/:householdId" element={<HouseholdDetailPage />} /></Routes></MemoryRouter>)
    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
    first.unmount()

    render(<MemoryRouter initialEntries={['/y/year-1/households/household-2']}><Routes><Route path="/y/:schoolYearId/households/:householdId" element={<HouseholdDetailPage />} /></Routes></MemoryRouter>)
    expect(await screen.findByRole('heading', { name: 'Stone second home' })).toBeInTheDocument()
    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
  })
})
