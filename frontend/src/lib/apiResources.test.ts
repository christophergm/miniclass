import { describe, expect, it } from 'vitest'

import { activeGradeLevels, activeHomerooms, type VocabularyResponse } from './apiResources'

const vocabulary: VocabularyResponse = {
  organization_id: 'org-test',
  homeroom_label: 'homeroom',
  grade_levels: [
    { id: 'g2', organization_id: 'org-test', code: '2', label: 'Second grade', ordinal: 2, created_at: '', updated_at: '' },
    { id: 'g1', organization_id: 'org-test', code: '1', label: 'First grade', ordinal: 1, created_at: '', updated_at: '' },
    { id: 'g0', organization_id: 'org-test', code: 'K', label: 'Kindergarten', ordinal: 0, retired_at: '2026-01-01', created_at: '', updated_at: '' },
  ],
  homerooms: [
    { id: 'h1', organization_id: 'org-test', name: 'Blue', created_at: '', updated_at: '' },
    { id: 'h2', organization_id: 'org-test', name: 'Green', retired_at: '2026-01-01', created_at: '', updated_at: '' },
  ],
}

describe('vocabulary picker helpers', () => {
  it('excludes retired entries and orders grades by their server ordinal', () => {
    expect(activeGradeLevels(vocabulary).map((grade) => grade.id)).toEqual(['g1', 'g2'])
    expect(activeHomerooms(vocabulary).map((homeroom) => homeroom.id)).toEqual(['h1'])
  })
})
