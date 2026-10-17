import { Link } from 'react-router-dom'
import { isLoggedIn } from '../lib/auth-storage'

export function HomePage() {
  const inSession = isLoggedIn()
  return (
    <div className="auth-layout">
      <div className="auth-card" style={{ maxWidth: 480 }}>
        <div className="brand">
          <div className="brand-mark">P</div>
          <div>
            <div className="brand-text">Pizza</div>
            <div className="brand-sub">BaaS Console</div>
          </div>
        </div>
        <h1>Backend infrastructure console</h1>
        <p className="lead">
          Sign in against the deployed auth-service. Manage account security now; organizations
          and API keys follow when the control-plane HTTP gateway is available.
        </p>
        <div className="row-actions">
          {inSession ? (
            <Link className="btn btn-primary" to="/app" style={{ textAlign: 'center' }}>
              Open console
            </Link>
          ) : (
            <>
              <Link className="btn btn-primary" to="/login" style={{ flex: 1, textAlign: 'center' }}>
                Sign in
              </Link>
              <Link className="btn btn-ghost" to="/register" style={{ flex: 1, textAlign: 'center' }}>
                Register
              </Link>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
