import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { ReactElement } from 'react'

/**
 * Renders inside a fresh QueryClient, so no test observes another's cache and a
 * call-count assertion measures only the render under test. `retry: false`
 * matches the client in main.tsx; a retrying client would turn one deliberate
 * API failure into several and stall the test.
 */
export function renderWithQueryClient(ui: ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>)
}
