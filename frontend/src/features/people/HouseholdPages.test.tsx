import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { householdApi } from './householdApi'
import { HouseholdDetailPage, HouseholdListPage } from './HouseholdPages'
import { peopleApi } from './api'
import type { Household } from './types'

const student = {
  id: 'student-1', school_year_id: 'year-1', legal_given_name: 'Riley', legal_family_name: 'Stone', display_name: 'Riley Stone', grade: '3', homeroom: 'A',
}

const otherStudent = { ...student, id: 'student-2', display_name: 'Jordan Reed', legal_given_name: 'Jordan', legal_family_name: 'Reed' }

const householdOne: Household = {
  id: 'household-1', school_year_id: 'year-1', display_name: 'Stone family', students: [student], adults: [],
}

const householdTwo: Household = {
  id: 'household-2', school_year_id: 'year-1', display_name: 'Stone second home', students: [student], adults: [],
}

afterEach(() => { vi.restoreAllMocks() })

describe('household pages', () => {
  it('lists households and keeps students grouped separately from adults', async () => {
    vi.spyOn(householdApi, 'list').mockResolvedValue([householdOne, householdTwo])

    render(<MemoryRouter initialEntries={['/y/year-1/households']}><Routes><Route path="/y/:schoolYearId/households" element={<HouseholdListPage />} /></Routes></MemoryRouter>)

    const table = await screen.findByRole('table', { name: 'Households' })
    expect(within(table).getByRole('link', { name: 'Stone family' })).toBeInTheDocument()
    expect(within(table).getByRole('link', { name: 'Stone second home' })).toBeInTheDocument()
    expect(within(table).getByText('Students')).toBeInTheDocument()
    expect(within(table).getByText('Adults')).toBeInTheDocument()
  })

  it('renders a student in a household detail and manages membership independently', async () => {
    vi.spyOn(householdApi, 'get').mockResolvedValue(householdOne)
    vi.spyOn(peopleApi, 'list').mockResolvedValue([student, otherStudent])
    const addMember = vi.spyOn(householdApi, 'addMember').mockResolvedValue()
    const removeMember = vi.spyOn(householdApi, 'removeMember').mockResolvedValue()

    render(<MemoryRouter initialEntries={['/y/year-1/households/household-1']}><Routes><Route path="/y/:schoolYearId/households/:householdId" element={<HouseholdDetailPage />} /></Routes></MemoryRouter>)

    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
    expect(screen.getByText(/Membership controls who is grouped with this household/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(removeMember).toHaveBeenCalledWith('year-1', 'household-1', 'student', 'student-1'))

    fireEvent.change(screen.getByLabelText('Person'), { target: { value: 'student-2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add member' }))
    await waitFor(() => expect(addMember).toHaveBeenCalledWith('year-1', 'household-1', 'student', 'student-2'))
  })

  it('renders a student belonging to two households in both household details', async () => {
    vi.spyOn(householdApi, 'get').mockImplementation(async (_schoolYearId, householdId) => householdId === 'household-1' ? householdOne : householdTwo)
    vi.spyOn(peopleApi, 'list').mockResolvedValue([])

    const first = render(<MemoryRouter initialEntries={['/y/year-1/households/household-1']}><Routes><Route path="/y/:schoolYearId/households/:householdId" element={<HouseholdDetailPage />} /></Routes></MemoryRouter>)
    expect(await screen.findByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
    first.unmount()

    render(<MemoryRouter initialEntries={['/y/year-1/households/household-2']}><Routes><Route path="/y/:schoolYearId/households/:householdId" element={<HouseholdDetailPage />} /></Routes></MemoryRouter>)
    expect(await screen.findByRole('heading', { name: 'Stone second home' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Riley Stone' })).toBeInTheDocument()
  })
})
