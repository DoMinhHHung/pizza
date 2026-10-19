import { FormEvent, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { ApiError, authApi } from '../api/client'
import {
  GlassyAuthShell,
  IconAt,
  IconLock,
  IconMail,
  IconUser,
} from '../components/GlassyAuthShell'
import { useAuth } from '../lib/auth-context'
import { setTokens } from '../lib/auth-storage'

type Mode = 'signin' | 'signup'

export function AuthGatePage() {
  const location = useLocation()
  const navigate = useNavigate()
  const { refreshUser } = useAuth()
  const initialMode: Mode = location.pathname.includes('register') ? 'signup' : 'signin'
  const [mode, setMode] = useState<Mode>(initialMode)

  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [signInError, setSignInError] = useState('')
  const [signInPending, setSignInPending] = useState(false)

  const [fullName, setFullName] = useState('')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [regPassword, setRegPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [signUpError, setSignUpError] = useState('')
  const [signUpSuccess, setSignUpSuccess] = useState('')
  const [signUpPending, setSignUpPending] = useState(false)

  function onModeChange(next: Mode) {
    setMode(next)
    navigate(next === 'signup' ? '/register' : '/login', { replace: true })
  }

  async function onSignIn(e: FormEvent) {
    e.preventDefault()
    setSignInError('')
    setSignInPending(true)
    try {
      const tokens = await authApi.login(identifier.trim(), password)
      setTokens(tokens.accessToken, tokens.refreshToken)
      await refreshUser()
      navigate('/app')
    } catch (err) {
      setSignInError(err instanceof ApiError ? err.message : 'Login failed')
    } finally {
      setSignInPending(false)
    }
  }

  async function onSignUp(e: FormEvent) {
    e.preventDefault()
    setSignUpError('')
    setSignUpSuccess('')
    if (regPassword !== confirm) {
      setSignUpError('Passwords do not match')
      return
    }
    setSignUpPending(true)
    try {
      const res = await authApi.register({
        email: email.trim(),
        username: username.trim().toLowerCase(),
        password: regPassword,
        fullName: fullName.trim(),
      })
      setSignUpSuccess(
        `Account created (${res.status}). Check your email to verify before signing in.`,
      )
    } catch (err) {
      setSignUpError(err instanceof ApiError ? err.message : 'Registration failed')
    } finally {
      setSignUpPending(false)
    }
  }

  const signIn = (
    <>
      <h2 className="g-title">Welcome Back</h2>
      <p className="g-lead">Enter your credentials to access your secure account</p>
      {signInError ? <div className="g-alert g-alert-error">{signInError}</div> : null}
      <form onSubmit={onSignIn}>
        <div className="g-field">
          <label htmlFor="si-identifier">Email or username</label>
          <div className="g-input-row">
            <IconMail />
            <input
              id="si-identifier"
              type="text"
              autoComplete="username"
              required
              placeholder="name@domain.com or username"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
            />
          </div>
        </div>
        <div className="g-field">
          <label htmlFor="si-password">Password</label>
          <div className="g-input-row">
            <IconLock />
            <input
              id="si-password"
              type={showPassword ? 'text' : 'password'}
              autoComplete="current-password"
              required
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <button
              type="button"
              className="g-toggle-pw"
              aria-label="Show password"
              onClick={() => setShowPassword((v) => !v)}
            >
              👁
            </button>
          </div>
        </div>
        <div className="g-row-between">
          <span />
          <Link className="g-link" to="/forgot-password">
            Forgot password?
          </Link>
        </div>
        <button className="g-btn-primary" type="submit" disabled={signInPending}>
          {signInPending ? 'Signing in…' : 'Sign In →'}
        </button>
      </form>
    </>
  )

  const signUp = (
    <>
      <h2 className="g-title">Create Account</h2>
      <p className="g-lead">Join Pizza to manage organizations and API keys</p>
      {signUpError ? <div className="g-alert g-alert-error">{signUpError}</div> : null}
      {signUpSuccess ? <div className="g-alert g-alert-success">{signUpSuccess}</div> : null}
      <form onSubmit={onSignUp}>
        <div className="g-field">
          <label htmlFor="su-name">Full name</label>
          <div className="g-input-row">
            <IconUser />
            <input
              id="su-name"
              type="text"
              required
              placeholder="Alex Johnson"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
            />
          </div>
        </div>
        <div className="g-field">
          <label htmlFor="su-username">Username</label>
          <div className="g-input-row">
            <IconAt />
            <input
              id="su-username"
              type="text"
              autoComplete="username"
              required
              minLength={3}
              maxLength={30}
              pattern="[a-zA-Z0-9_]+"
              title="3–30 characters: letters, numbers, underscore"
              placeholder="alex_j"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>
        </div>
        <div className="g-field">
          <label htmlFor="su-email">Email address</label>
          <div className="g-input-row">
            <IconMail />
            <input
              id="su-email"
              type="email"
              autoComplete="email"
              required
              placeholder="name@domain.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
        </div>
        <div className="g-row-2">
          <div className="g-field">
            <label htmlFor="su-password">Password</label>
            <div className="g-input-row">
              <IconLock />
              <input
                id="su-password"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                placeholder="••••••••"
                value={regPassword}
                onChange={(e) => setRegPassword(e.target.value)}
              />
            </div>
          </div>
          <div className="g-field">
            <label htmlFor="su-confirm">Confirm</label>
            <div className="g-input-row">
              <IconLock />
              <input
                id="su-confirm"
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                placeholder="••••••••"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
              />
            </div>
          </div>
        </div>
        <button className="g-btn-primary" type="submit" disabled={signUpPending}>
          {signUpPending ? 'Creating…' : 'Create Account →'}
        </button>
      </form>
    </>
  )

  return (
    <GlassyAuthShell mode={mode} onModeChange={onModeChange} signIn={signIn} signUp={signUp} />
  )
}
