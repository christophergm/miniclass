import { StrictMode } from 'react'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/lib/api'
import { resourceApi, type VocabularyResponse } from '@/lib/apiResources'
import { renderWithQueryClient } from '@/test/queryClient'

import { AdultDetailPage, AdultListPage, StudentDetailPage, StudentListPage } from './PeoplePages'
import { adultApi, guardianApi, studentApi, type Adult, type GuardianRelationship, type Student } from './roster'

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
const deletedStudent: Student = { ...ids, ...timestamps, id: 'student-4', legal_given_name: 'Sam', legal_family_name: 'Vale', display_name: 'Sam Vale', grade_level_id: 'grade-1', homeroom_id: 'homeroom-a', deleted_at: '2026-08-02T00:00:00Z' }

const vocabulary: VocabularyResponse = {
  school_year_id: 'year-1',
  homeroom_label: 'Homeroom',
  grade_levels: [
    { ...timestamps, id: 'grade-1', school_year_id: 'year-1', code: '1', label: 'First grade', ordinal: 1 },
    { ...timestamps, id: 'grade-2', school_year_id: 'year-1', code: '2', label: 'Second grade', ordinal: 2 },
  ],
  homerooms: [
    { ...timestamps, id: 'homeroom-a', school_year_id: 'year-1', name: 'Room A', external_identifier: null },
    { ...timestamps, id: 'homeroom-b', school_year_id: 'year-1', name: 'Room B', external_identifier: null },
    { ...timestamps, id: 'homeroom-c', school_year_id: 'year-1', name: 'Room C', external_identifier: null },
  ],
}

const adults: Adult[] = [
  { ...ids, ...timestamps, id: 'adult-1', legal_given_name: 'Morgan', legal_family_name: 'Lee', preferred_given_name: 'Mo', display_name: 'Mo Lee', email: 'mo@example.test', participation_intent: 'help' },
  { ...ids, ...timestamps, id: 'adult-2', legal_given_name: 'Jo', legal_family_name: 'Kim', display_name: 'Jo Kim', participation_intent: 'lead' },
]
const adult = adults[0]

function relationship(id: string, adult_id: string, student_id: string): GuardianRelationship {
  return { ...ids, ...timestamps, id, adult_id, student_id, relationship_type: 'parent' }
}

/**
 * The year's guardian edges arrive in one call, which is what the Guardians and
 * Children columns are derived from. The per-person reads are stubbed with the
 * same filter the server applies, so a detail page still sees only its own
 * person's edges; that a listing never reaches for them is asserted by call
 * count in the bounded-requests case below.
 */
function stubGuardianRelationships(relationships: GuardianRelationship[] = []) {
  vi.spyOn(guardianApi, 'listForYear').mockResolvedValue(relationships)
  vi.spyOn(guardianApi, 'listForStudent').mockImplementation(async (_schoolYearID, student_id) => relationships.filter((edge) => edge.student_id === student_id))
  vi.spyOn(guardianApi, 'listForAdult').mockImplementation(async (_schoolYearID, adult_id) => relationships.filter((edge) => edge.adult_id === adult_id))
}

// Every people surface now reads both rosters: its own, and the opposite one for
// the counterpart display names the derived column links to.
beforeEach(() => {
  vi.spyOn(resourceApi, 'getVocabulary').mockResolvedValue(vocabulary)
  vi.spyOn(studentApi, 'list').mockResolvedValue([])
  vi.spyOn(adultApi, 'list').mockResolvedValue([])
  stubGuardianRelationships()
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

function renderAdults(path = '/y/year-1/adults') {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/y/:schoolYearId/adults" element={<AdultListPage />} />
        <Route path="/y/:schoolYearId/adults/:personId" element={<AdultDetailPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

/** The person link is the first in each body row; the Guardians column adds more. */
function rowNames(table: HTMLElement) {
  return within(table).getAllByRole('row').slice(1).map((row) => within(row).getAllByRole('link')[0]?.textContent)
}

describe('people roster pages', () => {
  it('lists every guardian of a student, linked to the adult record', async () => {
    vi.spyOn(studentApi, 'list').mockResolvedValue([students[0]])
    vi.spyOn(adultApi, 'list').mockResolvedValue(adults)
    stubGuardianRelationships([relationship('rel-1', 'adult-1', 'student-2'), relationship('rel-2', 'adult-2', 'student-2')])

    renderStudents()

    const table = await screen.findByRole('table', { name: 'Students' })
    expect(within(table).getByRole('columnheader', { name: 'Guardians' })).toBeInTheDocument()
    // The link joins on the adult identifier the edge carries, never the name.
    expect(await within(table).findByRole('link', { name: 'Mo Lee' })).toHaveAttribute('href', '/y/year-1/adults/adult-1')
    expect(within(table).getByRole('link', { name: 'Jo Kim' })).toHaveAttribute('href', '/y/year-1/adults/adult-2')
  })

  it('lists the children an adult is a guardian of, linked to the student record', async () => {
    vi.spyOn(adultApi, 'list').mockResolvedValue([adult])
    vi.spyOn(studentApi, 'list').mockResolvedValue(students)
    stubGuardianRelationships([relationship('rel-1', 'adult-1', 'student-2'), relationship('rel-2', 'adult-1', 'student-1')])

    renderAdults()

    const table = await screen.findByRole('table', { name: 'Adults' })
    expect(within(table).getByRole('columnheader', { name: 'Children' })).toBeInTheDocument()
    expect(await within(table).findByRole('link', { name: 'Addie Zephyr' })).toHaveAttribute('href', '/y/year-1/students/student-2')
    expect(within(table).getByRole('link', { name: 'Bea Apple' })).toHaveAttribute('href', '/y/year-1/students/student-1')
  })

  // SPEC §8.2: nobody can be reached about a child with no guardian, so the
  // roster says so. SPEC §15.3 and ADR 0013 make an adult who is a guardian of
  // nobody an ordinary volunteer record, which is why only one of these two
  // columns carries a warning.
  it('warns on a student with no guardian and stays silent on an adult with no children', async () => {
    vi.spyOn(studentApi, 'list').mockResolvedValue([students[0]])
    vi.spyOn(adultApi, 'list').mockResolvedValue([adult])

    const studentList = renderStudents()
    const studentTable = await screen.findByRole('table', { name: 'Students' })
    expect(within(studentTable).getByText('No guardian')).toHaveAttribute('role', 'status')
    studentList.unmount()

    renderAdults()
    const adultTable = await screen.findByRole('table', { name: 'Adults' })
    expect(within(adultTable).queryByText('No guardian')).not.toBeInTheDocument()
    expect(within(adultTable).queryByRole('status')).not.toBeInTheDocument()
  })

  // SPEC §5.2 warns and never blocks: the record still saves.
  it('warns without blocking on the detail page of a student with no guardian', async () => {
    vi.spyOn(studentApi, 'get').mockResolvedValue(students[0])

    renderStudents('/y/year-1/students/student-2')

    expect(await screen.findByText('This student has no guardian yet. Nobody can be reached about this child. This is a warning only; you can still save the roster record.')).toBeInTheDocument()
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

  // The external identifier is an opaque source key: it matters when reconciling
  // one record against the export, and is noise in a roster scanned by name. It
  // stays on the detail page, where a single record is being worked on. Both
  // tables are asserted because they share the one PeopleTable.
  it('keeps the external identifier out of the roster tables but on the detail page', async () => {
    const identifiedStudent = { ...students[0], external_identifier: 'platform-9f3' }
    vi.spyOn(studentApi, 'list').mockResolvedValue([identifiedStudent])
    vi.spyOn(adultApi, 'list').mockResolvedValue([{ ...adult, external_identifier: 'platform-4c1' }])
    vi.spyOn(studentApi, 'get').mockResolvedValue(identifiedStudent)

    const studentListing = renderStudents()
    const studentTable = await screen.findByRole('table', { name: 'Students' })
    expect(within(studentTable).queryByRole('columnheader', { name: 'External ID' })).not.toBeInTheDocument()
    expect(within(studentTable).queryByText('platform-9f3')).not.toBeInTheDocument()
    studentListing.unmount()

    const adultListing = renderAdults()
    const adultTable = await screen.findByRole('table', { name: 'Adults' })
    expect(within(adultTable).queryByRole('columnheader', { name: 'External ID' })).not.toBeInTheDocument()
    expect(within(adultTable).queryByText('platform-4c1')).not.toBeInTheDocument()
    adultListing.unmount()

    renderStudents('/y/year-1/students/student-2')
    expect(await screen.findByLabelText('External identifier')).toHaveValue('platform-9f3')
  })

  // SPEC §10.1 makes grade ordinal and states the ordering is the definition's,
  // not the string's. Kindergarten sorts first on ordinal 0, while its label
  // would fall between "First grade" and "Second grade" alphabetically, so this
  // fails if the column ever orders by the rendered label.
  it('sorts students by grade ordinal rather than label, and puts a missing grade last', async () => {
    vi.spyOn(resourceApi, 'getVocabulary').mockResolvedValue({
      ...vocabulary,
      grade_levels: [
        { ...timestamps, id: 'grade-k', school_year_id: 'year-1', code: 'K', label: 'Kindergarten', ordinal: 0 },
        ...(vocabulary.grade_levels ?? []),
      ],
    })
    // All three graded students are needed to tell the orderings apart: by
    // ordinal it is Kindergarten, First, Second; by label it would be "First
    // grade", "Kindergarten", "Second grade".
    vi.spyOn(studentApi, 'list').mockResolvedValue([
      { ...students[1], id: 'student-second', display_name: 'Sec Ond', legal_family_name: 'Ond', grade_level_id: 'grade-2' },
      { ...students[1], id: 'student-none', display_name: 'No Grade', legal_family_name: 'Grade', grade_level_id: null },
      { ...students[1], id: 'student-k', display_name: 'Kin Der', legal_family_name: 'Der', grade_level_id: 'grade-k' },
      { ...students[1], id: 'student-first', display_name: 'Fir St', legal_family_name: 'St', grade_level_id: 'grade-1' },
    ])

    renderStudents()
    const table = await screen.findByRole('table', { name: 'Students' })
    await waitFor(() => expect(within(table).getByRole('button', { name: 'Grade' })).toBeInTheDocument())

    fireEvent.click(within(table).getByRole('button', { name: 'Grade' }))
    expect(rowNames(table)).toEqual(['Kin Der', 'Fir St', 'Sec Ond', 'No Grade'])
    expect(within(table).getByRole('columnheader', { name: 'Grade' })).toHaveAttribute('aria-sort', 'ascending')

    // Toggling brings the students needing a grade to the top rather than
    // burying them, which is the point of ordering by it at all.
    fireEvent.click(within(table).getByRole('button', { name: 'Grade' }))
    expect(rowNames(table)).toEqual(['No Grade', 'Sec Ond', 'Fir St', 'Kin Der'])
    expect(within(table).getByRole('columnheader', { name: 'Grade' })).toHaveAttribute('aria-sort', 'descending')
  })

  // SPEC §8.2 and §15.2 both state the intents as lead, help, unavailable.
  // A string compare would give help, lead, unavailable, so asserting lead
  // first fails if the order degrades to alphabetical.
  it('sorts adults by the spec order of participation intent, then by email with blanks last', async () => {
    vi.spyOn(adultApi, 'list').mockResolvedValue([
      { ...adult, id: 'adult-unavailable', display_name: 'Una Vail', legal_family_name: 'Vail', email: 'zoe@example.test', participation_intent: 'unavailable' },
      { ...adult, id: 'adult-none', display_name: 'Nora None', legal_family_name: 'None', email: undefined, participation_intent: null },
      { ...adult, id: 'adult-help', display_name: 'Hal Help', legal_family_name: 'Help', email: 'amy@example.test', participation_intent: 'help' },
      { ...adult, id: 'adult-lead', display_name: 'Lee Lead', legal_family_name: 'Lead', email: 'mia@example.test', participation_intent: 'lead' },
    ])

    renderAdults()
    const table = await screen.findByRole('table', { name: 'Adults' })
    await waitFor(() => expect(within(table).getByRole('button', { name: 'Participation' })).toBeInTheDocument())

    fireEvent.click(within(table).getByRole('button', { name: 'Participation' }))
    expect(rowNames(table)).toEqual(['Lee Lead', 'Hal Help', 'Una Vail', 'Nora None'])

    fireEvent.click(within(table).getByRole('button', { name: 'Email' }))
    expect(rowNames(table)).toEqual(['Hal Help', 'Lee Lead', 'Una Vail', 'Nora None'])
    expect(within(table).getByRole('columnheader', { name: 'Participation' })).toHaveAttribute('aria-sort', 'none')
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

  // Issue #104: the derived column used to be read one entity at a time, which
  // was ~181 requests on the reference program's roster (SPEC §3.2). It is now
  // the year's guardian edges in one request whatever the size of the year, so a
  // reintroduced fan-out fails here rather than passing quietly. The vocabulary
  // count differs by surface because only the student roster renders grade and
  // homeroom labels.
  it.each([
    ['student list', '/y/year-1/students', 'Mo Lee', 1, 1, 0, 0],
    ['adult list', '/y/year-1/adults', 'Addie Zephyr', 0, 1, 0, 0],
    ['student detail', '/y/year-1/students/student-2', 'Mo Lee', 1, 0, 1, 0],
    ['adult detail', '/y/year-1/adults/adult-1', 'Addie Zephyr', 0, 0, 0, 1],
  ])('reads the year’s guardian edges in a bounded number of requests on the %s', async (_surface, path, linkName, vocabularyCalls, yearWideCalls, perStudentCalls, perAdultCalls) => {
    const manyStudents: Student[] = [students[0], ...Array.from({ length: 180 }, (_value, index) => ({
      ...ids, ...timestamps, id: `student-extra-${index}`, legal_given_name: 'Sam', legal_family_name: `Extra${index}`, display_name: `Sam Extra${index}`, grade_level_id: 'grade-1', homeroom_id: 'homeroom-a',
    }))]
    stubGuardianRelationships([relationship('rel-1', 'adult-1', 'student-2')])
    vi.spyOn(studentApi, 'list').mockResolvedValue(manyStudents)
    vi.spyOn(studentApi, 'get').mockResolvedValue(students[0])
    vi.spyOn(adultApi, 'list').mockResolvedValue([adult])
    vi.spyOn(adultApi, 'get').mockResolvedValue(adult)

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

    expect(await screen.findByRole('link', { name: linkName })).toBeInTheDocument()
    expect(guardianApi.listForYear).toHaveBeenCalledTimes(yearWideCalls)
    // A listing renders 181 people and still asks nothing per person.
    expect(guardianApi.listForStudent).toHaveBeenCalledTimes(perStudentCalls)
    expect(guardianApi.listForAdult).toHaveBeenCalledTimes(perAdultCalls)
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
