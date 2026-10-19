import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { GlassyAuthShell, GlassySimpleCard } from './GlassyAuthShell'

describe('GlassyAuthShell', () => {
  it('renders brand and active sign-in layer', () => {
    render(
      <MemoryRouter>
        <GlassyAuthShell
          mode="signin"
          onModeChange={() => {}}
          signIn={<h2 className="g-title">Welcome Back</h2>}
          signUp={<h2 className="g-title">Create Account</h2>}
        />
      </MemoryRouter>,
    )
    expect(screen.getByText('Pizza')).toBeInTheDocument()
    expect(screen.getByText('Console')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /welcome back/i })).toBeInTheDocument()
  })
})

describe('GlassySimpleCard', () => {
  it('renders title and children', () => {
    render(
      <MemoryRouter>
        <GlassySimpleCard title="Reset password" lead="Paste token">
          <button type="button">Submit</button>
        </GlassySimpleCard>
      </MemoryRouter>,
    )
    expect(screen.getByRole('heading', { name: /reset password/i })).toBeInTheDocument()
    expect(screen.getByText(/paste token/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /submit/i })).toBeInTheDocument()
  })
})
