import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { authApi } from '../api/client'
import { useAuth } from '../lib/auth-context'

export function Shell() {
  const { user, setUser } = useAuth()
  const navigate = useNavigate()

  async function logout() {
    await authApi.logout()
    setUser(null)
    navigate('/login')
  }

  const initial = (user?.fullName || user?.email || '?').charAt(0).toUpperCase()

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand" style={{ marginBottom: 0 }}>
          <div className="brand-mark">P</div>
          <div className="brand-text">Pizza</div>
        </div>
        <nav className="topbar-nav">
          <NavLink to="/app" end>
            Overview
          </NavLink>
          <NavLink to="/app/account">Account</NavLink>
          <NavLink to="/app/orgs">Organizations</NavLink>
        </nav>
        <div className="user-chip">
          <div className="avatar">{initial}</div>
          <span>{user?.fullName || user?.email}</span>
          <button type="button" className="btn btn-ghost" onClick={logout}>
            Log out
          </button>
        </div>
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
