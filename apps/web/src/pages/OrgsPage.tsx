import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError } from '../api/client'
import { controlApi, type Organization } from '../api/control'

export function OrgsPage() {
  const [orgs, setOrgs] = useState<Organization[]>([])
  const [error, setError] = useState('')
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [pending, setPending] = useState(false)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setError('')
    try {
      setOrgs(await controlApi.listOrgs())
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to load organizations')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError('')
    try {
      await controlApi.createOrg(name.trim(), slug.trim())
      setName('')
      setSlug('')
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Create failed')
    } finally {
      setPending(false)
    }
  }

  return (
    <div>
      <h1 style={{ marginBottom: 8 }}>Organizations</h1>
      <p className="lead">Create and open an organization to manage members, projects, and API keys.</p>
      {error ? <div className="alert alert-error">{error}</div> : null}
      <div className="grid-2">
        <div className="card">
          <h2>Your organizations</h2>
          {loading ? (
            <p className="card-desc">Loading…</p>
          ) : orgs.length === 0 ? (
            <div className="empty">No organizations yet.</div>
          ) : (
            <div className="stack">
              {orgs.map((o) => (
                <Link
                  key={o.id}
                  to={`/app/orgs/${o.id}`}
                  style={{
                    display: 'block',
                    padding: '12px 14px',
                    borderRadius: 8,
                    border: '1px solid var(--border)',
                    color: 'inherit',
                    background: 'var(--bg-elevated)',
                  }}
                >
                  <div style={{ fontWeight: 600 }}>{o.name}</div>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4 }}>
                    {o.slug} · <span className="badge badge-muted">{o.role}</span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
        <div className="card">
          <h2>Create organization</h2>
          <p className="card-desc">POST /api/v1/orgs — you become OWNER</p>
          <form onSubmit={onCreate} className="stack">
            <div className="field">
              <label htmlFor="name">Name</label>
              <input id="name" required value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="field">
              <label htmlFor="slug">Slug</label>
              <input
                id="slug"
                required
                pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
                placeholder="acme-inc"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
              />
            </div>
            <button className="btn btn-primary" type="submit" disabled={pending}>
              {pending ? 'Creating…' : 'Create'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
