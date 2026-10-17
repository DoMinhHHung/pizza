import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../api/client'
import {
  controlApi,
  type ApiKey,
  type Member,
  type Project,
} from '../api/control'

export function OrgDetailPage() {
  const { orgId = '' } = useParams()
  const [members, setMembers] = useState<Member[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedProject, setSelectedProject] = useState<string>('')
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [secretOnce, setSecretOnce] = useState('')

  const [memberUserId, setMemberUserId] = useState('')
  const [memberRole, setMemberRole] = useState('MEMBER')
  const [projectName, setProjectName] = useState('')
  const [projectSlug, setProjectSlug] = useState('')
  const [keyName, setKeyName] = useState('')

  const load = useCallback(async () => {
    if (!orgId) return
    setError('')
    try {
      const [m, p] = await Promise.all([
        controlApi.listMembers(orgId),
        controlApi.listProjects(orgId),
      ])
      setMembers(m)
      setProjects(p)
      if (p.length && !selectedProject) {
        setSelectedProject(p[0].id)
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to load org')
    }
  }, [orgId, selectedProject])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    if (!selectedProject) {
      setKeys([])
      return
    }
    controlApi
      .listApiKeys(selectedProject)
      .then(setKeys)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load keys'))
  }, [selectedProject])

  async function onAddMember(e: FormEvent) {
    e.preventDefault()
    try {
      await controlApi.addMember(orgId, memberUserId.trim(), memberRole)
      setMemberUserId('')
      setInfo('Member added')
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Add member failed')
    }
  }

  async function onCreateProject(e: FormEvent) {
    e.preventDefault()
    try {
      const p = await controlApi.createProject(orgId, projectName.trim(), projectSlug.trim())
      setProjectName('')
      setProjectSlug('')
      setSelectedProject(p.id)
      setInfo('Project created')
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Create project failed')
    }
  }

  async function onCreateKey(e: FormEvent) {
    e.preventDefault()
    if (!selectedProject) return
    try {
      const res = await controlApi.createApiKey(selectedProject, keyName.trim())
      setKeyName('')
      setSecretOnce(res.secret)
      setInfo('API key created — copy the secret now, it is shown once')
      setKeys(await controlApi.listApiKeys(selectedProject))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Create key failed')
    }
  }

  async function revoke(id: string) {
    try {
      await controlApi.revokeApiKey(id)
      if (selectedProject) setKeys(await controlApi.listApiKeys(selectedProject))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Revoke failed')
    }
  }

  async function changeRole(userId: string, role: string) {
    try {
      await controlApi.updateMemberRole(orgId, userId, role)
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Update role failed')
    }
  }

  async function removeMember(userId: string) {
    try {
      await controlApi.removeMember(orgId, userId)
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Remove failed')
    }
  }

  return (
    <div>
      <p style={{ marginBottom: 8 }}>
        <Link to="/app/orgs">← Organizations</Link>
      </p>
      <h1 style={{ marginBottom: 8 }}>Organization</h1>
      <p className="lead" style={{ fontFamily: 'var(--mono)', fontSize: 12 }}>
        {orgId}
      </p>
      {error ? <div className="alert alert-error">{error}</div> : null}
      {info ? <div className="alert alert-success">{info}</div> : null}
      {secretOnce ? (
        <div className="alert alert-info">
          Secret (copy once): <code style={{ fontFamily: 'var(--mono)' }}>{secretOnce}</code>
        </div>
      ) : null}

      <div className="grid-2" style={{ marginTop: 8 }}>
        <div className="card">
          <h2>Members</h2>
          <div className="stack" style={{ marginBottom: 16 }}>
            {members.map((m) => (
              <div
                key={m.userId}
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  gap: 8,
                  alignItems: 'center',
                  fontSize: 13,
                }}
              >
                <span className="meta-value">{m.userId}</span>
                <select
                  value={m.role}
                  onChange={(e) => void changeRole(m.userId, e.target.value)}
                  style={{
                    background: 'var(--bg-elevated)',
                    color: 'var(--text)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    padding: '4px 8px',
                  }}
                >
                  <option value="OWNER">OWNER</option>
                  <option value="ADMIN">ADMIN</option>
                  <option value="MEMBER">MEMBER</option>
                </select>
                <button type="button" className="btn btn-danger" onClick={() => void removeMember(m.userId)}>
                  Remove
                </button>
              </div>
            ))}
          </div>
          <form onSubmit={onAddMember} className="stack">
            <div className="field">
              <label>User ID (UUID from auth)</label>
              <input required value={memberUserId} onChange={(e) => setMemberUserId(e.target.value)} />
            </div>
            <div className="field">
              <label>Role</label>
              <select
                value={memberRole}
                onChange={(e) => setMemberRole(e.target.value)}
                style={{
                  width: '100%',
                  padding: '11px 14px',
                  borderRadius: 8,
                  border: '1px solid var(--border)',
                  background: 'var(--bg-elevated)',
                  color: 'var(--text)',
                }}
              >
                <option value="MEMBER">MEMBER</option>
                <option value="ADMIN">ADMIN</option>
                <option value="OWNER">OWNER</option>
              </select>
            </div>
            <button className="btn btn-primary" type="submit">
              Add member
            </button>
          </form>
        </div>

        <div className="card">
          <h2>Projects</h2>
          <div className="stack" style={{ marginBottom: 16 }}>
            {projects.map((p) => (
              <button
                key={p.id}
                type="button"
                className="btn btn-ghost"
                style={{
                  justifyContent: 'flex-start',
                  borderColor: selectedProject === p.id ? 'var(--accent)' : undefined,
                }}
                onClick={() => setSelectedProject(p.id)}
              >
                {p.name} <span style={{ color: 'var(--text-muted)', marginLeft: 8 }}>{p.slug}</span>
              </button>
            ))}
          </div>
          <form onSubmit={onCreateProject} className="stack">
            <div className="field">
              <label>Name</label>
              <input required value={projectName} onChange={(e) => setProjectName(e.target.value)} />
            </div>
            <div className="field">
              <label>Slug</label>
              <input required value={projectSlug} onChange={(e) => setProjectSlug(e.target.value)} />
            </div>
            <button className="btn btn-primary" type="submit">
              Create project
            </button>
          </form>
        </div>
      </div>

      <div className="card" style={{ marginTop: 20 }}>
        <h2>API keys</h2>
        <p className="card-desc">
          {selectedProject ? `Project ${selectedProject}` : 'Select a project first'}
        </p>
        {selectedProject ? (
          <>
            <div className="stack" style={{ marginBottom: 16 }}>
              {keys.length === 0 ? (
                <div className="empty">No API keys</div>
              ) : (
                keys.map((k) => (
                  <div
                    key={k.id}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      gap: 12,
                      alignItems: 'center',
                      fontSize: 13,
                    }}
                  >
                    <div>
                      <strong>{k.name}</strong>
                      <div className="meta-value">
                        {k.keyPrefix}… {k.revokedAt ? '(revoked)' : ''}
                      </div>
                    </div>
                    {!k.revokedAt ? (
                      <button type="button" className="btn btn-danger" onClick={() => void revoke(k.id)}>
                        Revoke
                      </button>
                    ) : (
                      <span className="badge badge-muted">revoked</span>
                    )}
                  </div>
                ))
              )}
            </div>
            <form onSubmit={onCreateKey} className="stack" style={{ maxWidth: 360 }}>
              <div className="field">
                <label>Key name</label>
                <input required value={keyName} onChange={(e) => setKeyName(e.target.value)} />
              </div>
              <button className="btn btn-primary" type="submit">
                Create API key
              </button>
            </form>
          </>
        ) : null}
      </div>
    </div>
  )
}
