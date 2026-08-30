import { fireEvent, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '@/lib/api'
import type { ImportPreview } from '@/lib/apiResources'
import { renderWithQueryClient } from '@/test/queryClient'

import { ImportPage } from './ImportPage'
import { useCommitImport, usePreviewImport } from './useImports'

vi.mock('./useImports', () => ({
  useCommitImport: vi.fn(),
  usePreviewImport: vi.fn(),
}))

const emptyCounts = { create: 0, update: 0, unchanged: 0, conflict: 0, error: 0 }

function preview(overrides: Partial<ImportPreview> = {}): ImportPreview {
  return {
    kind: 'roster_json',
    school_year_id: 'year-1',
    content_hash: 'abc123',
    rows: [],
    guardian_relationship_removals: [],
    exclusions: [],
    warnings: [],
    counts: emptyCounts,
    ...overrides,
  }
}

function renderImport() {
  return renderWithQueryClient(
    <MemoryRouter initialEntries={['/y/year-1/imports']}>
      <Routes>
        <Route element={<ImportPage />} path="/y/:schoolYearId/imports" />
      </Routes>
    </MemoryRouter>,
  )
}

function setup(previewResult = preview(), commitResult = preview()) {
  const previewMutation = { mutate: vi.fn(), reset: vi.fn(), isPending: false, isError: false, error: null }
  const commitMutation = { mutate: vi.fn(), reset: vi.fn(), isPending: false, isError: false, error: null }
  previewMutation.mutate.mockImplementation((_variables, options) => options.onSuccess(previewResult))
  commitMutation.mutate.mockImplementation((_variables, options) => options.onSuccess(commitResult))
  vi.mocked(usePreviewImport).mockReturnValue(previewMutation as never)
  vi.mocked(useCommitImport).mockReturnValue(commitMutation as never)
  return { previewMutation, commitMutation }
}

beforeEach(() => vi.clearAllMocks())

describe('ImportPage', () => {
  it('blocks commit while an Error record is present and explains why', () => {
    const result = preview({ counts: { ...emptyCounts, error: 1 } })
    setup(result)
    renderImport()

    fireEvent.change(screen.getByLabelText('Document'), { target: { files: [new File(['source'], 'roster.json', { type: 'application/json' })] } })
    fireEvent.click(screen.getByRole('button', { name: 'Preview file' }))

    expect(screen.getByRole('alert')).toHaveTextContent('Commit is disabled because 1 record is in Error')
    expect(screen.getByRole('button', { name: 'Commit import' })).toBeDisabled()
  })

  it('puts guardian removals in their own clearly labelled section', () => {
    const result = preview({
      guardian_relationship_removals: [{ existing_id: 'edge-1', adult_external_identifier: 'adult-1', student_external_identifier: 'student-1', relationship_type: 'parent', detail: 'The adult row omitted this child.' }],
    })
    setup(result)
    renderImport()

    fireEvent.change(screen.getByLabelText('Document'), { target: { files: [new File(['source'], 'roster.json')] } })
    fireEvent.click(screen.getByRole('button', { name: 'Preview file' }))

    expect(screen.getByRole('heading', { name: 'Guardian relationship removals' })).toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'Guardian relationship removals' })).toHaveTextContent('adult-1')
    expect(screen.getByText('The adult row omitted this child.')).toBeInTheDocument()
  })

  it('carries the preview hash into commit and presents a hash mismatch clearly', () => {
    const result = preview()
    const { commitMutation } = setup(result)
    const errorMutation = commitMutation as { isError: boolean; error: unknown }
    errorMutation.isError = true
    errorMutation.error = new ApiError('http', 'the submitted document does not match the reviewed preview content hash', 409, 'import-invalid')
    commitMutation.mutate.mockImplementation(() => undefined)
    renderImport()

    const file = new File(['source'], 'roster.json')
    fireEvent.change(screen.getByLabelText('Document'), { target: { files: [file] } })
    fireEvent.click(screen.getByRole('button', { name: 'Preview file' }))
    fireEvent.click(screen.getByRole('button', { name: 'Commit import' }))

    expect(commitMutation.mutate).toHaveBeenCalledWith({ kind: 'roster_json', schoolYearId: 'year-1', document: file, contentHash: 'abc123' }, expect.anything())
    expect(screen.getByRole('alert')).toHaveTextContent('file changed — preview it again')
  })

  it('shows committed counts and links to the import audit entry', () => {
    const result = preview({ counts: { ...emptyCounts, create: 2 } })
    setup(preview(), result)
    renderImport()

    const file = new File(['source'], 'roster.json')
    fireEvent.change(screen.getByLabelText('Document'), { target: { files: [file] } })
    fireEvent.click(screen.getByRole('button', { name: 'Preview file' }))
    fireEvent.click(screen.getByRole('button', { name: 'Commit import' }))

    expect(screen.getByRole('heading', { name: 'Import committed' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'View the import audit entry' })).toHaveAttribute('href', '/audit-log?object_type=import')
    expect(screen.getByText('2')).toBeInTheDocument()
  })
})
