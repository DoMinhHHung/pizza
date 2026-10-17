import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { authApi } from '../api/client'
import { useAuth } from '../lib/auth-context'

export function OverviewPage() {
  const { user } = useAuth()
  const [health, setHealth] = useState<string>('…')

  useEffect(() => {
    authApi
      .health()
      .then((h) => setHealth(h.status))
      .catch(() => setHealth('DOWN'))
  }, [])

  return (
    <div>
      <h1 style={{ marginBottom: 8 }}>Overview</h1>
      <p className="lead" style={{ marginBottom: 24 }}>
        Auth session is live. Manage <Link to="/app/orgs">organizations</Link> via control-plane HTTP
        API.
      </p>
      <div className="grid-2">
        <div className="card">
          <h2>Session</h2>
          <p className="card-desc">GET /api/v1/auth/me</p>
          <div className="meta-row">
            <span className="meta-label">Name</span>
            <span>{user?.fullName}</span>
          </div>
          <div className="meta-row">
            <span className="meta-label">Email</span>
            <span className="meta-value">{user?.email}</span>
          </div>
          <div className="meta-row">
            <span className="meta-label">Status</span>
            <span className={user?.status === 'ACTIVE' ? 'badge badge-ok' : 'badge badge-warn'}>
              {user?.status}
            </span>
          </div>
          <div className="meta-row">
            <span className="meta-label">User ID</span>
            <span className="meta-value">{user?.id}</span>
          </div>
        </div>
        <div className="card">
          <h2>Services</h2>
          <div className="meta-row">
            <span className="meta-label">Auth health</span>
            <span className={health === 'UP' ? 'badge badge-ok' : 'badge badge-warn'}>{health}</span>
          </div>
          <div className="meta-row">
            <span className="meta-label">Auth URL</span>
            <span className="meta-value">{import.meta.env.VITE_AUTH_URL}</span>
          </div>
          <div className="meta-row">
            <span className="meta-label">Control URL</span>
            <span className="meta-value">{import.meta.env.VITE_CONTROL_URL || 'http://localhost:8081'}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
