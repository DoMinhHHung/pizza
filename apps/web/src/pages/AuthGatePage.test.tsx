import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AuthProvider } from '../lib/auth-context'
import { clearTokens } from '../lib/auth-storage'
import { AuthGatePage } from './AuthGatePage'

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<typeof import('../api/client')>('../api/client')
  return {
    ...actual,
    authApi: {
      ...actual.authApi,
      login: vi.fn(),
      register: vi.fn(),
      me: vi.fn(),
    },
  }
})

function renderGate(path = '/login') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<AuthGatePage />} />
          <Route path="/register" element={<AuthGatePage />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('AuthGatePage', () => {
  beforeEach(() => {
    clearTokens()
  })

  it('renders sign-in form on /login', () => {
    renderGate('/login')
    expect(screen.getByRole('heading', { name: /welcome back/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/email or username/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in →/i })).toBeInTheDocument()
  })

  it('renders sign-up form on /register', () => {
    renderGate('/register')
    expect(screen.getByRole('heading', { name: /create account/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/^username$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/email address/i)).toBeInTheDocument()
  })

  it('can switch to sign-up via footer link', async () => {
    const user = userEvent.setup()
    renderGate('/login')
    await user.click(screen.getByRole('button', { name: /^sign up$/i }))
    expect(await screen.findByRole('heading', { name: /create account/i })).toBeInTheDocument()
  })
})
