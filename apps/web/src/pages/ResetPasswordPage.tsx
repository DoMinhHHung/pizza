import { FormEvent, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { ApiError, authApi } from '../api/client'
import { AuthCard } from '../components/AuthCard'

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const [token, setToken] = useState(params.get('token') || '')
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
      const res = await authApi.resetPassword(token.trim(), password)
      setSuccess(res.message)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Reset failed')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthCard title="Reset password" lead="Paste the token from your email if it is not in the URL.">
      {error ? <div className="alert alert-error">{error}</div> : null}
      {success ? (
        <div className="alert alert-success">
          {success} <Link to="/login">Sign in</Link>
        </div>
      ) : null}
      <form onSubmit={onSubmit}>
        <div className="field">
          <label htmlFor="token">Token</label>
          <input id="token" required value={token} onChange={(e) => setToken(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="password">New password</label>
          <input
            id="password"
            type="password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? 'Saving…' : 'Update password'}
        </button>
      </form>
    </AuthCard>
  )
}
