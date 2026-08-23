import { ApiError } from '../../lib/api'
import { useHealth } from '../../lib/hooks/useHealth'

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) {
    return timestamp
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function titleCase(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1)
}

function HealthMetric({ label, value, detail }: { label: string; value: string; detail?: string }) {
  return (
    <div className="health-metric">
      <span>{label}</span>
      <strong>{value}</strong>
      {detail && <small>{detail}</small>}
    </div>
  )
}

export function HealthCheck() {
  const { data, error, isError, isFetching, isLoading, refetch } = useHealth()

  if (isLoading) {
    return (
      <div className="page-content health-page">
        <p className="eyebrow accent">System health</p>
        <h2>Backend health check</h2>
        <div className="health-card health-loading" role="status" aria-live="polite">
          <span className="health-spinner" aria-hidden="true" />
          <div>
            <strong>Checking backend health…</strong>
            <p>Connecting to the MiniClass API.</p>
          </div>
        </div>
      </div>
    )
  }

  if (isError || !data) {
    const message = error instanceof ApiError ? error.message : 'An unexpected error occurred.'

    return (
      <div className="page-content health-page">
        <p className="eyebrow accent">System health</p>
        <h2>Backend health check</h2>
        <div className="health-card health-error" role="alert">
          <div className="health-status-icon error" aria-hidden="true">!</div>
          <div className="health-message">
            <strong>Backend health check failed</strong>
            <p>{message}</p>
            <button className="secondary-button" type="button" onClick={() => void refetch()} disabled={isFetching}>
              {isFetching ? 'Trying again…' : 'Try again'}
            </button>
          </div>
        </div>
      </div>
    )
  }

  const isHealthy = data.status === 'healthy'

  return (
    <div className="page-content health-page">
      <div className="health-heading">
        <div>
          <p className="eyebrow accent">System health</p>
          <h2>Backend health check</h2>
          <p className="health-intro">A live view of the MiniClass API and its database connection.</p>
        </div>
        <button className="refresh-button" type="button" onClick={() => void refetch()} disabled={isFetching}>
          <span aria-hidden="true">↻</span>
          {isFetching ? 'Refreshing…' : 'Refresh now'}
        </button>
      </div>

      <div className={`health-card health-summary ${isHealthy ? 'healthy' : 'unhealthy'}`}>
        <div className={`health-status-icon ${isHealthy ? 'success' : 'warning'}`} aria-hidden="true">
          {isHealthy ? '✓' : '!'}
        </div>
        <div className="health-message">
          <div className="health-status-line">
            <strong>{isHealthy ? 'All systems operational' : 'Backend needs attention'}</strong>
            <span className={`status-pill ${isHealthy ? 'success' : 'warning'}`}>{titleCase(data.status)}</span>
          </div>
          <p>The latest health check completed successfully.</p>
        </div>
      </div>

      <div className="health-metrics" aria-label="Backend health details">
        <HealthMetric label="Database" value={titleCase(data.database)} detail="Connection status" />
        <HealthMetric label="Version" value={data.version} detail="Running release" />
        <HealthMetric label="Last checked" value={formatTimestamp(data.timestamp)} detail="Backend timestamp" />
      </div>

      <p className="health-refresh-note" aria-live="polite">
        Automatically refreshes every 30 seconds.
        {isFetching && <span> Checking now…</span>}
      </p>
    </div>
  )
}
