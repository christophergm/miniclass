import { StrictMode } from 'react'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/lib/api'
import { resourceApi, type VocabularyResponse } from '@/lib/apiResources'
import { renderWithQueryClient } from '@/test/queryClient'

import { AdultDetailPage, AdultListPage, StudentDetailPage, StudentListPage } from './PeoplePages'
import { adultApi, guardianApi, householdApi, studentApi, type Adult, type Household, type Student } from './roster'

// Fixtures are typed against the generated contract, so a backend field rename
// fails this file at compile time rather than passing against a shape the API
// never returns (ADR 0004).

const timestamps = { created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z' }
const ids = { organization_id: 'org-1', school_year_id: 'year-1' }

const students: Student[] = [
  { ...ids, ...timestamps, id: 'student-2', legal_given_name: 'Ada', legal_family_name: 'Zephyr', preferred_given_name: 'Addie', display_name: 'Addie Zephyr', grade_level_id: 'grade-2', homeroom_id: 'homeroom-c' },
  { ...ids, ...timestamps, id: 'student-1', legal_given_name: 'Bea', legal_family_name: 'Apple', display_name: 'Bea Apple', grade_level_id: 'grade-1', homeroom_id: 'homeroom-a' },
  { ...ids, ...timestamps, id: 'student-3', legal_given_name: 'Ari', legal_family_name: 'Apple', preferred_given_name: 'Aria', display_name: 'Aria Apple', grade_level_id: 'grade-2', homeroom_id: 'homeroom-b' },
]

const vocabulary: VocabularyResponse = {
  organization_id: 'org-1',
  homeroom_label: 'Homeroom',
  grade_levels: [
    { ...timestamps, id: 'grade-1', organization_id: 'org-1', code: '1', label: 'First grade', ordinal: 1 },
    { ...timestamps, id: 'grade-2', organization_id: 'org-1', code: '2', label: 'Second grade', ordinal: 2 },
  ],
  homerooms: [
    { ...timestamps, id: 'homeroom-a', organization_id: 'org-1', name: 'Room A' },
    { ...timestamps, id: 'homeroom-b', organization_id: 'org-1', name: 'Room B' },
    { ...timestamps, id: 'homeroom-c', organization_id: 'org-1', name: 'Room C' },
  ],
}

const deletedStudent: Student = { ...ids, ...timestamps, id: 'student-4', legal_given_name: 'Sam', legal_family_name: 'Vale', display_name: 'Sam Vale', grade_level_id: 'grade-1', homeroom_id: 'homeroom-a', deleted_at: '2026-08-02T00:00:00Z' }

const adult: Adult = { ...ids, ...timestamps, id: 'adult-1', legal_given_name: 'Morgan', legal_family_name: 'Lee', preferred_given_name: 'Mo', display_name: 'Mo Lee', email: 'mo@example.test', participation_intent: 'help' }

const households: Household[] = [
  { ...ids, ...timestamps, id: 'household-1', display_name: 'Primary home' },
  { ...ids, ...timestamps, id: 'household-2', display_name: 'Second home' },
]

/**
 * Membership arrives as the year's rows in one call. The per-household
 * sub-resources are stubbed to reject so that a reintroduced fan-out fails here
 * rather than passing quietly.
 */
function stubMembership(
  studentIdsByHousehold: Record<string, string[]> = {},
  adultIdsByHousehold: Record<string, string[]> = {},
  householdList: Household[] = households,
) {
  vi.spyOn(householdApi, 'list').mockResolvedValue(householdList)
  vi.spyOn(householdApi, 'listMembership').mockResolvedValue({
    students: Object.entries(studentIdsByHousehold).flatMap(([household_id, studentIds]) =>
      studentIds.map((student_id) => ({ id: `${household_id}-${student_id}`, household_id, student_id }))),
    adults: Object.entries(adultIdsByHousehold).flatMap(([household_id, adultIds]) =>
      adultIds.map((adult_id) => ({ id: `${household_id}-${adult_id}`, household_id, adult_id }))),
  })
  vi.spyOn(householdApi, 'listStudents').mockRejectedValue(new Error('a roster surface must not read membership one household at a time'))
  vi.spyOn(householdApi, 'listAdults').mockRejectedValue(new Error('a roster surface must not read membership one household at a time'))
}

beforeEach(() => {
  vi.spyOn(resourceApi, 'getVocabulary').mockResolvedValue(vocabulary)
  stubMembership()
})

afterEach(() => { vi.restoreAllMocks() })

function renderStudents(path = '/y/year-1/students') {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/y/:schoolYearId/students" element={<StudentListPage />} />
        <Route path="/y/:schoolYearId/students/:personId" element={<StudentDetailPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('people roster pages', () => {
  it('shows every household for a student in the student list', async () => {
    vi.spyOn(studentApi, 'list').mockResolvedValue([students[0]])
    stubMembership({ 'household-1': ['student-2'], 'household-2': ['student-2'] })

    renderStudents()

    const table = await screen.findByRole('table', { name: 'Students' })
    expect(await within(table).findByRole('link', { name: 'Primary home' })).toBeInTheDocument()
    expect(within(table).getByRole('link', { name: 'Second home' })).toBeInTheDocument()
  })

  it('shows both household links on student detail and warns without blocking a student with none', async () => {
    vi.spyOn(studentApi, 'get').mockResolvedValue(students[0])
    vi.spyOn(adultApi, 'list').mockResolvedValue([])
    vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([])
    stubMembership({ 'household-1': ['student-2'], 'household-2': ['student-2'] })

    const detail = renderStudents('/y/year-1/students/student-2')

    expect(await screen.findByRole('link', { name: 'Primary home' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Second home' })).toBeInTheDocument()

    detail.unmount()
    vi.restoreAllMocks()
    vi.spyOn(resourceApi, 'getVocabulary').mockResolvedValue(vocabulary)
    vi.spyOn(studentApi, 'get').mockResolvedValue(students[0])
    vi.spyOn(adultApi, 'list').mockResolvedValue([])
    vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([])
    stubMembership()

    renderStudents('/y/year-1/students/student-2')
    expect(await screen.findByText('This person has no household yet. This is a warning only; you can still save the roster record.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
  })

  it('uses the API display name and sorts by legal family then given name', async () => {
    vi.spyOn(studentApi, 'list').mockResolvedValue(students)

    renderStudents()

    const table = await screen.findByRole('table', { name: 'Students' })
    const rows = within(table).getAllByRole('row')
    expect(within(rows[1]).getAllByRole('link')[0]).toHaveTextContent('Aria Apple')
    expect(within(rows[2]).getAllByRole('link')[0]).toHaveTextContent('Bea Apple')
    expect(within(rows[3]).getAllByRole('link')[0]).toHaveTextContent('Addie Zephyr')
    expect(screen.queryByText('Ada Zephyr')).not.toBeInTheDocument()
  })

  // SPEC §21.3. The deleted filter, the deleted-row treatment and the restore
  // action were removed in #96 because they were typed against fields the API
  // did not return. These assertions are on the request the filter issues and
  // the call the action makes, so they fail if the controls go inert again.
  it('asks for deleted students only when the filter is set, and restores one with a reason', async () => {
    const list = vi.spyOn(studentApi, 'list').mockImplementation(async (_schoolYearId, includeDeleted) =>
      includeDeleted ? [students[1], deletedStudent] : [students[1]])
    const restore = vi.spyOn(studentApi, 'restore').mockResolvedValue({ ...deletedStudent, deleted_at: undefined })

    renderStudents()

    const table = await screen.findByRole('table', { name: 'Students' })
    expect(within(table).queryByText('Sam Vale')).not.toBeInTheDocument()
    expect(list).toHaveBeenCalledWith('year-1', false)

    fireEvent.click(screen.getByLabelText('Show deleted'))

    expect(await screen.findByText('Sam Vale')).toBeInTheDocument()
    expect(list).toHaveBeenCalledWith('year-1', true)

    const deletedRow = screen.getByText('Sam Vale').closest('tr')!
    expect(deletedRow.className).toContain('text-muted-foreground')
    expect(within(screen.getByText('Bea Apple').closest('tr')!).queryByRole('button', { name: 'Restore' })).not.toBeInTheDocument()

    // SPEC §5.4: the reason is the record, so an abandoned prompt is not a
    // restore.
    vi.spyOn(window, 'prompt').mockReturnValueOnce(null)
    fireEvent.click(within(deletedRow).getByRole('button', { name: 'Restore' }))
    expect(restore).not.toHaveBeenCalled()

    vi.spyOn(window, 'prompt').mockReturnValueOnce('deleted by mistake')
    fireEvent.click(within(deletedRow).getByRole('button', { name: 'Restore' }))
    await waitFor(() => expect(restore).toHaveBeenCalledWith('year-1', 'student-4', 'deleted by mistake'))
  })

  it('renders the grade and homeroom labels for the identifiers the roster returns', async () => {
    vi.spyOn(studentApi, 'list').mockResolvedValue([students[0]])

    renderStudents()

    const table = await screen.findByRole('table', { name: 'Students' })
    expect(await within(table).findByText('Second grade')).toBeInTheDocument()
    expect(within(table).getByText('Room C')).toBeInTheDocument()
    expect(within(table).queryByText('grade-2')).not.toBeInTheDocument()
  })

  it('filters students by search, grade, and homeroom', async () => {
    vi.spyOn(studentApi, 'list').mockResolvedValue(students)

    renderStudents()
    await screen.findByRole('table', { name: 'Students' })
    await waitFor(() => expect(screen.getByLabelText('Grade')).toContainHTML('Second grade'))

    fireEvent.change(screen.getByLabelText('Search by name'), { target: { value: 'aria' } })
    expect(screen.getByRole('link', { name: 'Aria Apple' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Bea Apple' })).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Search by name'), { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('Grade'), { target: { value: 'grade-2' } })
    fireEvent.change(screen.getByLabelText('Homeroom'), { target: { value: 'homeroom-c' } })
    expect(screen.getByRole('link', { name: 'Addie Zephyr' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Aria Apple' })).not.toBeInTheDocument()
  })

  // SPEC §3.2 records ~90 households in the reference program. The Households
  // column used to cost one request per household on each of these surfaces.
  // The vocabulary count differs by surface because only the student roster
  // renders grade and homeroom labels.
  it.each([
    ['student list', '/y/year-1/students', 1],
    ['adult list', '/y/year-1/adults', 0],
    ['student detail', '/y/year-1/students/student-2', 1],
    ['adult detail', '/y/year-1/adults/adult-1', 0],
  ])('reads household membership in a bounded number of requests on the %s', async (_surface, path, vocabularyCalls) => {
    const manyHouseholds: Household[] = Array.from({ length: 90 }, (_value, index) => ({
      ...ids, ...timestamps, id: `household-${index}`, display_name: `Household ${index}`,
    }))
    stubMembership({ 'household-7': ['student-2'] }, { 'household-7': ['adult-1'] }, manyHouseholds)
    vi.spyOn(studentApi, 'list').mockResolvedValue([students[0]])
    vi.spyOn(studentApi, 'get').mockResolvedValue(students[0])
    vi.spyOn(adultApi, 'list').mockResolvedValue([adult])
    vi.spyOn(adultApi, 'get').mockResolvedValue(adult)
    vi.spyOn(guardianApi, 'listForStudent').mockResolvedValue([])
    vi.spyOn(guardianApi, 'listForAdult').mockResolvedValue([])

    // StrictMode is on in main.tsx and deliberately double-invokes effects in
    // development. React Query dedupes the second mount; the useEffect fetches
    // these pages used to run did not, so every read went out twice.
    renderWithQueryClient(
      <StrictMode>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route path="/y/:schoolYearId/students" element={<StudentListPage />} />
            <Route path="/y/:schoolYearId/students/:personId" element={<StudentDetailPage />} />
            <Route path="/y/:schoolYearId/adults" element={<AdultListPage />} />
            <Route path="/y/:schoolYearId/adults/:personId" element={<AdultDetailPage />} />
          </Routes>
        </MemoryRouter>
      </StrictMode>,
    )

    expect(await screen.findByRole('link', { name: 'Household 7' })).toBeInTheDocument()
    expect(householdApi.listMembership).toHaveBeenCalledTimes(1)
    expect(householdApi.list).toHaveBeenCalledTimes(1)
    expect(householdApi.listStudents).not.toHaveBeenCalled()
    expect(householdApi.listAdults).not.toHaveBeenCalled()
    expect(resourceApi.getVocabulary).toHaveBeenCalledTimes(vocabularyCalls)
  })

  it('shows adult participation intent in the adult list', async () => {
    vi.spyOn(adultApi, 'list').mockResolvedValue([adult])

    renderWithQueryClient(
      <MemoryRouter initialEntries={['/y/year-1/adults']}>
        <Routes><Route path="/y/:schoolYearId/adults" element={<AdultListPage />} /></Routes>
      </MemoryRouter>,
    )

    const table = await screen.findByRole('table', { name: 'Adults' })
    expect(within(table).getByText('Mo Lee')).toBeInTheDocument()
    expect(within(table).getByText('help')).toBeInTheDocument()
  })

  it('sends the grade and homeroom identifiers the contract requires, not their labels', async () => {
    const create = vi.spyOn(studentApi, 'create').mockResolvedValue(students[0])

    renderStudents('/y/year-1/students/new')
    await waitFor(() => expect(screen.getByLabelText('Grade')).toContainHTML('Second grade'))

    fireEvent.change(screen.getByLabelText('Legal given name'), { target: { value: 'Ada' } })
    fireEvent.change(screen.getByLabelText('Legal family name'), { target: { value: 'Zephyr' } })
    fireEvent.change(screen.getByLabelText('Grade'), { target: { value: 'grade-2' } })
    fireEvent.change(screen.getByLabelText('Homeroom'), { target: { value: 'homeroom-c' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(create).toHaveBeenCalledWith('year-1', {
      legal_given_name: 'Ada',
      legal_family_name: 'Zephyr',
      grade_level_id: 'grade-2',
      homeroom_id: 'homeroom-c',
    }))
  })

  it('renders server field errors inline without client-side validation', async () => {
    vi.spyOn(studentApi, 'create').mockRejectedValue(new ApiError('http', 'Please correct the highlighted fields.', 422, 'validation-error', [
      { location: 'body.legal_given_name', message: 'Legal given name is required.' },
    ]))

    renderStudents('/y/year-1/students/new')
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(screen.getByText('Legal given name is required.')).toBeInTheDocument())
    expect(screen.getAllByRole('alert')[0]).toHaveTextContent('Please correct the highlighted fields.')
  })
})
