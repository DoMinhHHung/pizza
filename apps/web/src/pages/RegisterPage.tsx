import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, authApi } from '../api/client'
import { AuthCard } from '../components/AuthCard'

export function RegisterPage() {
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSuccess('')
    setPending(true)
    try {
      const res = await authApi.register({
        email: email.trim(),
        password,
        fullName: fullName.trim(),
      })
      setSuccess(
        `Account created (${res.status}). Check your email to verify before signing in.`,
      )
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Registration failed')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthCard title="Create account" lead="Register with email. Verification is required before login.">
      {error ? <div className="alert alert-error">{error}</div> : null}
      {success ? <div className="alert alert-success">{success}</div> : null}
      <form onSubmit={onSubmit}>
        <div className="field">
          <label htmlFor="fullName">Full name</label>
          <input
            id="fullName"
            required
            value={fullName}
            onChange={(e) => setFullName(e.target.value)}
          />
        </div>
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
            autoComplete="new-password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? 'Creating…' : 'Create account'}
        </button>
      </form>
      <div className="auth-footer">
        Already have an account? <Link to="/login">Sign in</Link>
      </div>
    </AuthCard>
  )
}
