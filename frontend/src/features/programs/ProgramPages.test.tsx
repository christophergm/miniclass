import { fireEvent, screen } from '@testing-library/react'
import { MemoryRouter, Outlet, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { SchoolYear } from '@/lib/apiResources'
import { renderWithQueryClient } from '@/test/queryClient'

import { SessionPage } from './ProgramPages'

const mocks = vi.hoisted(() => ({
  transition: vi.fn(),
  sessionState: 'planning',
}))

vi.mock('./usePrograms', () => {
  const query = (data: unknown) => vi.fn(() => ({ data, isLoading: false, isError: false, error: null }))
  const mutation = (mutate = vi.fn()) => vi.fn(() => ({ mutate, isPending: false, isError: false, error: null }))
  return {
    useSession: vi.fn(() => ({ data: { id: 'session-1', organization_id: 'org-1', school_year_id: 'year-1', program_id: 'program-1', name: 'Autumn session', ordinal: 1, state: mocks.sessionState, draft_assignments_stale: false, meeting_dates: ['2026-10-02'], feasibility_warnings: [], created_at: '', updated_at: '' }, isLoading: false, isError: false, error: null })),
    usePrograms: query([{ id: 'program-1', organization_id: 'org-1', school_year_id: 'year-1', name: 'Enrichment', created_at: '', updated_at: '' }]),
    useMeetingDates: query([{ id: 'date-1', school_year_id: 'year-1', organization_id: 'org-1', program_id: 'program-1', session_id: 'session-1', meeting_date: '2026-10-02', created_at: '', updated_at: '' }]),
    useOfferings: query([{ id: 'offering-1', school_year_id: 'year-1', organization_id: 'org-1', program_id: 'program-1', session_id: 'session-1', name: 'Making', description: 'Build a project', capacity: 10, minimum_viable_enrollment: 2, min_grade_level_id: 'grade-1', max_grade_level_id: 'grade-1', location: 'Studio', meeting_point: 'Front desk', meeting_instructions: 'Ask for the key', interest_area_id: null, created_at: '', updated_at: '' }]),
    useCatalogFeasibility: query({ participant_count: 2, warnings: [{ id: 'capacity', severity: 'warning', message: 'Capacity is below participation.', participant_count: 2, total_capacity: 1, total_minimum_viable_enrollment: 0, shortfall: 1, affected_grades: [], affected_areas: [], offering_ids: [] }] }),
    useProgramInterestAreas: query([{ id: 'area-1', label: 'Making', ordinal: 1, retired_at: null }]),
    useVocabulary: query({ school_year_id: 'year-1', grade_levels: [{ id: 'grade-1', school_year_id: 'year-1', code: '1', label: 'Grade 1', ordinal: 1, created_at: '', updated_at: '' }], homerooms: [] }),
    useProgramMemberships: query([{ id: 'membership-1', student_id: 'student-1', legal_given_name: 'Riley', legal_family_name: 'Synthetic', grade_missing: false }]),
    useSessionNonParticipations: query([]),
    useSessionObjectiveWeights: query({ defaults: { repeat_offering_penalty: 10 }, effective: { repeat_offering_penalty: 10 }, overrides: {} }),
    useCreateMeetingDate: mutation(), useUpdateMeetingDate: mutation(), useDeleteMeetingDate: mutation(),
    useCreateOffering: mutation(), useUpdateOffering: mutation(), useDeleteOffering: mutation(),
    useTransitionSession: mutation(mocks.transition), useCreateSessionNonParticipation: mutation(),
    useUpdateSessionNonParticipation: mutation(), useDeleteSessionNonParticipation: mutation(),
    useUpdateSession: mutation(), useUpdateSessionObjectiveWeights: mutation(),
    useCreateProgram: mutation(), useMissingGradeCount: query({ missing_grade_count: 0 }), useCreateInterestArea: mutation(),
    useAddProgramMembership: mutation(), useRemoveProgramMembership: mutation(), useCreateSession: mutation(),
    useProgramObjectiveWeights: query(undefined), useUpdateProgramObjectiveWeights: mutation(), useReorderInterestAreas: mutation(),
  }
})

vi.mock('@/lib/hooks/useVocabulary', () => ({ useVocabulary: vi.fn(() => ({ data: { school_year_id: 'year-1', grade_levels: [{ id: 'grade-1', school_year_id: 'year-1', code: '1', label: 'Grade 1', ordinal: 1, created_at: '', updated_at: '' }], homerooms: [] }, isLoading: false, isError: false, error: null })) }))
vi.mock('@/features/people/roster-queries', () => ({ usePeople: vi.fn(() => ({ data: [], isLoading: false, isError: false, error: null })) }))

const year = (state: SchoolYear['state']): SchoolYear => ({ id: 'year-1', organization_id: 'org-1', label: '2026–27', state, created_at: '', updated_at: '' })

function renderSession(currentYear = year('active')) {
  function ContextRoute() { return <Outlet context={currentYear} /> }
  return renderWithQueryClient(<MemoryRouter initialEntries={['/y/year-1/programs/program-1/sessions/session-1']}><Routes><Route element={<ContextRoute />} path="/y/:schoolYearId"><Route element={<SessionPage />} path="programs/:programId/sessions/:sessionId" /></Route></Routes></MemoryRouter>)
}

describe('SessionPage', () => {
  beforeEach(() => { mocks.transition.mockReset(); mocks.sessionState = 'planning' })

  it('consolidates the authoring surfaces and makes feasibility warnings visibly non-blocking', () => {
    renderSession()

    expect(screen.getByRole('heading', { name: 'Autumn session' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Meeting dates' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Offerings' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Session non-participation' })).toBeInTheDocument()
    expect(screen.getByText('Advisory only — you can continue authoring.')).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Choose allowed state' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Catalog Published' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Complete' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Transition' })).toBeInTheDocument()
  })

  it('applies a transition directly when no confirmation is required', () => {
    renderSession()

    fireEvent.change(screen.getByRole('combobox', { name: 'Next session state' }), { target: { value: 'catalog_published' } })
    fireEvent.click(screen.getByRole('button', { name: 'Transition' }))

    expect(mocks.transition).toHaveBeenCalledWith(
      { state: 'catalog_published', reason: undefined, confirm: false },
      expect.any(Object),
    )
  })

  it('uses the preview and confirmation flow for a backward transition', () => {
    mocks.sessionState = 'voting_closed'
    mocks.transition.mockImplementation((value, options) => {
      if (!value.confirm) options.onSuccess({ requires_confirmation: true, warnings: [] })
    })
    renderSession()

    fireEvent.change(screen.getByRole('combobox', { name: 'Next session state' }), { target: { value: 'voting_open' } })
    expect(screen.getByRole('button', { name: 'Preview Transition…' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Complete' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Preview Transition…' }))
    expect(screen.getByRole('textbox', { name: 'Transition reason' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm transition' })).toBeDisabled()

    fireEvent.change(screen.getByRole('textbox', { name: 'Transition reason' }), { target: { value: 'Reopen for correction' } })
    fireEvent.click(screen.getByRole('button', { name: 'Confirm transition' }))
    expect(mocks.transition).toHaveBeenLastCalledWith(
      { state: 'voting_open', reason: 'Reopen for correction', confirm: true },
      expect.any(Object),
    )
  })

  it('renders every mutation control disabled for a closed year', () => {
    renderSession(year('closed'))

    expect(screen.getByRole('heading', { name: 'Read-only history' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create offering' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Add date' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Transition' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Mark not participating' })).toBeDisabled()
  })
})
