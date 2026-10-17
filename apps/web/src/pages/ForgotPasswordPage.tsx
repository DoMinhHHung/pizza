import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, authApi } from '../api/client'
import { AuthCard } from '../components/AuthCard'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSuccess('')
    setPending(true)
    try {
      const res = await authApi.forgotPassword(email.trim())
      setSuccess(res.message)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Request failed')
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthCard title="Forgot password" lead="We will email a reset link if the account exists.">
      {error ? <div className="alert alert-error">{error}</div> : null}
      {success ? <div className="alert alert-success">{success}</div> : null}
      <form onSubmit={onSubmit}>
        <div className="field">
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? 'Sending…' : 'Send reset link'}
        </button>
      </form>
      <div className="auth-footer">
        <Link to="/login">Back to sign in</Link>
      </div>
    </AuthCard>
  )
}
