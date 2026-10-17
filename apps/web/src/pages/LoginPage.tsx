import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiError, authApi } from '../api/client'
import { setTokens } from '../lib/auth-storage'
import { useAuth } from '../lib/auth-context'
import { AuthCard } from '../components/AuthCard'

export function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  const navigate = useNavigate()
  const { refreshUser } = useAuth()

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setPending(true)
    try {
      const tokens = await authApi.login(email.trim(), password)
      setTokens(tokens.accessToken, tokens.refreshToken)
      await refreshUser()
      navigate('/app')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Login failed')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthCard title="Sign in" lead="Access your Pizza organizations and API keys.">
      {error ? <div className="alert alert-error">{error}</div> : null}
      <form onSubmit={onSubmit}>
        <div className="field">
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
        <div className="field">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
      <div className="auth-footer">
        <Link to="/forgot-password">Forgot password?</Link>
        <br />
        No account? <Link to="/register">Create one</Link>
      </div>
    </AuthCard>
  )
}
