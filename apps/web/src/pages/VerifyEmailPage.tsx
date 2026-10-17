import { FormEvent, useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { ApiError, authApi } from '../api/client'
import { AuthCard } from '../components/AuthCard'

export function VerifyEmailPage() {
  const [params] = useSearchParams()
  const [token, setToken] = useState(params.get('token') || '')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [pending, setPending] = useState(false)

  async function verify(value: string) {
    setError('')
    setSuccess('')
    setPending(true)
    try {
      const res = await authApi.verifyEmail(value.trim())
      setSuccess(res.message)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Verification failed')
    } finally {
      setPending(false)
    }
  }

  useEffect(() => {
    const t = params.get('token')
    if (t) {
      void verify(t)
    }
  }, [params])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    await verify(token)
  }

  return (
    <AuthCard title="Verify email" lead="Confirm your address to activate the account.">
      {error ? <div className="alert alert-error">{error}</div> : null}
      {success ? (
        <div className="alert alert-success">
          {success} <Link to="/login">Sign in</Link>
        </div>
      ) : null}
      <form onSubmit={onSubmit}>
        <div className="field">
          <label htmlFor="token">Verification token</label>
          <input id="token" required value={token} onChange={(e) => setToken(e.target.value)} />
        </div>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? 'Verifying…' : 'Verify'}
        </button>
      </form>
    </AuthCard>
  )
}
