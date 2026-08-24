import { Navigate, Route, Routes } from 'react-router-dom'

import { HealthCheck } from '@/features/health/HealthCheck'

function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate replace to="/health" />} />
      <Route path="/health" element={<HealthCheck />} />
      <Route path="*" element={<Navigate replace to="/health" />} />
    </Routes>
  )
}

export default App
