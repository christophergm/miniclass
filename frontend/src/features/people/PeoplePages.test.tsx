import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { PeopleApiError, peopleApi } from './api'
import { AdultListPage, StudentDetailPage, StudentListPage } from './PeoplePages'
import type { Adult } from './types'

const students = [
  {
    id: 'student-2', school_year_id: 'year-1', legal_given_name: 'Ada', legal_family_name: 'Zephyr', preferred_given_name: 'Addie', display_name: 'Addie Zephyr', grade: '2', homeroom: 'C',
  },
  {
    id: 'student-1', school_year_id: 'year-1', legal_given_name: 'Bea', legal_family_name: 'Apple', preferred_given_name: null, display_name: 'Bea Apple', grade: '1', homeroom: 'A',
  },
  {
    id: 'student-3', school_year_id: 'year-1', legal_given_name: 'Ari', legal_family_name: 'Apple', preferred_given_name: 'Aria', display_name: 'Aria Apple', grade: '2', homeroom: 'B',
  },
]

afterEach(() => { vi.restoreAllMocks() })

function renderStudents(path = '/y/year-1/students') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/y/:schoolYearId/students" element={<StudentListPage />} />
        <Route path="/y/:schoolYearId/students/:personId" element={<StudentDetailPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('people roster pages', () => {
  it('uses the API display name and sorts by legal family then given name', async () => {
    vi.spyOn(peopleApi, 'list').mockResolvedValue(students)

    renderStudents()

    const table = await screen.findByRole('table', { name: 'Students' })
    const rows = within(table).getAllByRole('row')
    expect(within(rows[1]).getByRole('link')).toHaveTextContent('Aria Apple')
    expect(within(rows[2]).getByRole('link')).toHaveTextContent('Bea Apple')
    expect(within(rows[3]).getByRole('link')).toHaveTextContent('Addie Zephyr')
    expect(screen.queryByText('Ada Zephyr')).not.toBeInTheDocument()
  })

  it('filters students by search, grade, and homeroom', async () => {
    vi.spyOn(peopleApi, 'list').mockResolvedValue(students)

    renderStudents()
    await screen.findByRole('table', { name: 'Students' })

    fireEvent.change(screen.getByLabelText('Search by name'), { target: { value: 'aria' } })
    expect(screen.getByRole('link', { name: 'Aria Apple' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Bea Apple' })).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Search by name'), { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('Grade'), { target: { value: '2' } })
    fireEvent.change(screen.getByLabelText('Homeroom'), { target: { value: 'C' } })
    expect(screen.getByRole('link', { name: 'Addie Zephyr' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Aria Apple' })).not.toBeInTheDocument()
  })

  it('shows adult participation intent in the adult list', async () => {
    vi.spyOn(peopleApi, 'list').mockResolvedValue([{
      id: 'adult-1', school_year_id: 'year-1', legal_given_name: 'Morgan', legal_family_name: 'Lee', preferred_given_name: 'Mo', display_name: 'Mo Lee', email: 'mo@example.test', phone: null, participation_intent: 'help',
    } as Adult])

    render(
      <MemoryRouter initialEntries={['/y/year-1/adults']}>
        <Routes><Route path="/y/:schoolYearId/adults" element={<AdultListPage />} /></Routes>
      </MemoryRouter>,
    )

    const table = await screen.findByRole('table', { name: 'Adults' })
    expect(within(table).getByText('Mo Lee')).toBeInTheDocument()
    expect(within(table).getByText('help')).toBeInTheDocument()
  })

  it('renders server field errors inline without client-side validation', async () => {
    vi.spyOn(peopleApi, 'create').mockRejectedValue(new PeopleApiError('Please correct the highlighted fields.', 400, { legal_given_name: 'Legal given name is required.' }))

    renderStudents('/y/year-1/students/new')
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(screen.getByText('Legal given name is required.')).toBeInTheDocument())
    expect(screen.getAllByRole('alert')[0]).toHaveTextContent('Please correct the highlighted fields.')
  })
})
