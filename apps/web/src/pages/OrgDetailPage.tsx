import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ApiError, authApi } from '../api/client'
import {
  controlApi,
  type ApiKey,
  type Member,
  type Organization,
  type Project,
} from '../api/control'
import { useAuth } from '../lib/auth-context'

export function OrgDetailPage() {
  const { orgId = '' } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()

  const [org, setOrg] = useState<Organization | null>(null)
  const [members, setMembers] = useState<Member[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [selectedProject, setSelectedProject] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(true)

  const [inviteQ, setInviteQ] = useState('')
  const [memberRole, setMemberRole] = useState('MEMBER')
  const [projectName, setProjectName] = useState('')
  const [projectSlug, setProjectSlug] = useState('')
  const [keyName, setKeyName] = useState('')
  const [lastSecret, setLastSecret] = useState('')

  const [orgName, setOrgName] = useState('')
  const [orgSlug, setOrgSlug] = useState('')
  const [editProjectName, setEditProjectName] = useState('')
  const [editProjectSlug, setEditProjectSlug] = useState('')

  const canManage = org?.role === 'OWNER' || org?.role === 'ADMIN'
  const isOwner = org?.role === 'OWNER'

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError('')
    try {
      const [o, m, p] = await Promise.all([
        controlApi.getOrg(orgId),
        controlApi.listMembers(orgId),
        controlApi.listProjects(orgId),
      ])
      setOrg(o)
      setOrgName(o.name)
      setOrgSlug(o.slug)
      setMembers(m)
      setProjects(p)
      if (p.length && !selectedProject) setSelectedProject(p[0].id)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to load organization')
    } finally {
      setLoading(false)
    }
  }, [orgId, selectedProject])

  useEffect(() => {
    void load()
  }, [orgId])

  useEffect(() => {
    if (!selectedProject) {
      setKeys([])
      return
    }
    const p = projects.find((x) => x.id === selectedProject)
    if (p) {
      setEditProjectName(p.name)
      setEditProjectSlug(p.slug)
    }
    controlApi
      .listApiKeys(selectedProject)
      .then(setKeys)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Failed to load keys'))
  }, [selectedProject, projects])

  async function onInvite(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSuccess('')
    try {
      const found = await authApi.lookup(inviteQ.trim())
      await controlApi.addMember(orgId, found.id, memberRole)
      setSuccess(`Added ${found.username || found.email} as ${memberRole}`)
      setInviteQ('')
      setMembers(await controlApi.listMembers(orgId))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Invite failed')
    }
  }

  async function onCreateProject(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const p = await controlApi.createProject(orgId, projectName.trim(), projectSlug.trim())
      setProjects(await controlApi.listProjects(orgId))
      setSelectedProject(p.id)
      setProjectName('')
      setProjectSlug('')
      setSuccess(`Project ${p.slug} created`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Create project failed')
    }
  }

  async function onUpdateProject(e: FormEvent) {
    e.preventDefault()
    if (!selectedProject) return
    setError('')
    try {
      await controlApi.updateProject(selectedProject, editProjectName.trim(), editProjectSlug.trim())
      setProjects(await controlApi.listProjects(orgId))
      setSuccess('Project updated')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Update project failed')
    }
  }

  async function onDeleteProject() {
    if (!selectedProject || !confirm('Delete this project and its API keys?')) return
    setError('')
    try {
      await controlApi.deleteProject(selectedProject)
      const p = await controlApi.listProjects(orgId)
      setProjects(p)
      setSelectedProject(p[0]?.id || '')
      setSuccess('Project deleted')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Delete project failed')
    }
  }

  async function onCreateKey(e: FormEvent) {
    e.preventDefault()
    if (!selectedProject) return
    setError('')
    setLastSecret('')
    try {
      const res = await controlApi.createApiKey(selectedProject, keyName.trim())
      setLastSecret(res.secret)
      setKeyName('')
      setKeys(await controlApi.listApiKeys(selectedProject))
      setSuccess('API key created — copy the secret now, it will not be shown again')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Create key failed')
    }
  }

  async function onRevokeKey(id: string) {
    if (!confirm('Revoke this API key?')) return
    try {
      await controlApi.revokeApiKey(id)
      if (selectedProject) setKeys(await controlApi.listApiKeys(selectedProject))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Revoke failed')
    }
  }

  async function onRole(userId: string, role: string) {
    try {
      await controlApi.updateMemberRole(orgId, userId, role)
      setMembers(await controlApi.listMembers(orgId))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Update role failed')
    }
  }

  async function onRemove(userId: string) {
    if (!confirm('Remove this member?')) return
    try {
      await controlApi.removeMember(orgId, userId)
      setMembers(await controlApi.listMembers(orgId))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Remove failed')
    }
  }

  async function onUpdateOrg(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const o = await controlApi.updateOrg(orgId, orgName.trim(), orgSlug.trim())
      setOrg(o)
      setSuccess('Organization updated')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Update org failed')
    }
  }

  async function onDeleteOrg() {
    if (!confirm('Delete this organization permanently?')) return
    try {
      await controlApi.deleteOrg(orgId)
      navigate('/app/orgs')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Delete org failed')
    }
  }

  if (loading) return <p className="card-desc">Loading organization…</p>

  return (
    <div className="stack">
      <div>
        <Link to="/app/orgs" style={{ fontSize: 13 }}>
          ← Organizations
        </Link>
        <h1 style={{ margin: '8px 0 4px', fontSize: 22 }}>
          {org?.name || 'Organization'}
        </h1>
        <p className="card-desc" style={{ margin: 0 }}>
          {org?.slug} · your role: <strong>{org?.role}</strong>
        </p>
      </div>

      {error ? <div className="alert alert-error">{error}</div> : null}
      {success ? <div className="alert alert-success">{success}</div> : null}

      {isOwner ? (
        <div className="card">
          <h2>Organization settings</h2>
          <form onSubmit={onUpdateOrg} className="stack">
            <div className="field">
              <label htmlFor="org-name">Name</label>
              <input id="org-name" required value={orgName} onChange={(e) => setOrgName(e.target.value)} />
            </div>
            <div className="field">
              <label htmlFor="org-slug">Slug</label>
              <input id="org-slug" required value={orgSlug} onChange={(e) => setOrgSlug(e.target.value)} />
            </div>
            <div className="row-actions">
              <button className="btn btn-primary" type="submit">
                Save
              </button>
              <button className="btn btn-danger" type="button" onClick={() => void onDeleteOrg()}>
                Delete organization
              </button>
            </div>
          </form>
        </div>
      ) : null}

      <div className="grid-2">
        <div className="card">
          <h2>Members</h2>
          <p className="card-desc">Invite by username or email (lookup via auth-service).</p>
          {canManage ? (
            <form onSubmit={onInvite} className="stack" style={{ marginBottom: 16 }}>
              <div className="field">
                <label htmlFor="invite">Username or email</label>
                <input
                  id="invite"
                  required
                  placeholder="alex or alex@example.com"
                  value={inviteQ}
                  onChange={(e) => setInviteQ(e.target.value)}
                />
              </div>
              <div className="field">
                <label htmlFor="role">Role</label>
                <select id="role" value={memberRole} onChange={(e) => setMemberRole(e.target.value)}>
                  <option value="MEMBER">MEMBER</option>
                  <option value="ADMIN">ADMIN</option>
                  {isOwner ? <option value="OWNER">OWNER</option> : null}
                </select>
              </div>
              <button className="btn btn-primary" type="submit">
                Invite member
              </button>
            </form>
          ) : null}
          <div className="stack">
            {members.map((m) => (
              <div key={m.userId} className="meta-row" style={{ alignItems: 'center' }}>
                <div>
                  <div className="meta-value" style={{ textAlign: 'left' }}>
                    {m.userId === user?.id ? `${m.userId} (you)` : m.userId}
                  </div>
                  <span className="badge badge-muted">{m.role}</span>
                </div>
                {canManage && m.userId !== user?.id ? (
                  <div className="row-actions">
                    {isOwner ? (
                      <select value={m.role} onChange={(e) => void onRole(m.userId, e.target.value)}>
                        <option value="MEMBER">MEMBER</option>
                        <option value="ADMIN">ADMIN</option>
                        <option value="OWNER">OWNER</option>
                      </select>
                    ) : null}
                    <button className="btn btn-ghost" type="button" onClick={() => void onRemove(m.userId)}>
                      Remove
                    </button>
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <h2>Projects</h2>
          {canManage ? (
            <form onSubmit={onCreateProject} className="stack" style={{ marginBottom: 16 }}>
              <div className="field">
                <label htmlFor="pname">Name</label>
                <input id="pname" required value={projectName} onChange={(e) => setProjectName(e.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="pslug">Slug</label>
                <input id="pslug" required value={projectSlug} onChange={(e) => setProjectSlug(e.target.value)} />
              </div>
              <button className="btn btn-primary" type="submit">
                Create project
              </button>
            </form>
          ) : null}
          <div className="stack">
            {projects.length === 0 ? <p className="empty">No projects yet</p> : null}
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
                {p.name}{' '}
                <span style={{ color: 'var(--text-muted)', marginLeft: 8 }}>{p.slug}</span>
              </button>
            ))}
          </div>
        </div>
      </div>

      {selectedProject ? (
        <div className="card">
          <h2>Selected project</h2>
          {canManage ? (
            <form onSubmit={onUpdateProject} className="stack" style={{ marginBottom: 16 }}>
              <div className="field">
                <label htmlFor="epname">Name</label>
                <input
                  id="epname"
                  required
                  value={editProjectName}
                  onChange={(e) => setEditProjectName(e.target.value)}
                />
              </div>
              <div className="field">
                <label htmlFor="epslug">Slug</label>
                <input
                  id="epslug"
                  required
                  value={editProjectSlug}
                  onChange={(e) => setEditProjectSlug(e.target.value)}
                />
              </div>
              <div className="row-actions">
                <button className="btn btn-primary" type="submit">
                  Update project
                </button>
                <button className="btn btn-danger" type="button" onClick={() => void onDeleteProject()}>
                  Delete project
                </button>
              </div>
            </form>
          ) : null}

          <h2 style={{ marginTop: 8 }}>API keys</h2>
          <p className="card-desc">Secrets are shown only once at creation.</p>
          {lastSecret ? (
            <div className="alert alert-info">
              Secret: <code style={{ wordBreak: 'break-all' }}>{lastSecret}</code>
            </div>
          ) : null}
          {canManage ? (
            <form onSubmit={onCreateKey} className="stack" style={{ marginBottom: 16 }}>
              <div className="field">
                <label htmlFor="kname">Key name</label>
                <input id="kname" required value={keyName} onChange={(e) => setKeyName(e.target.value)} />
              </div>
              <button className="btn btn-primary" type="submit">
                Create API key
              </button>
            </form>
          ) : null}
          <div className="stack">
            {keys.length === 0 ? <p className="empty">No API keys</p> : null}
            {keys.map((k) => (
              <div key={k.id} className="meta-row">
                <div>
                  <strong>{k.name}</strong>
                  <div className="meta-value" style={{ textAlign: 'left' }}>
                    {k.keyPrefix}… {k.revokedAt ? '(revoked)' : ''}
                  </div>
                </div>
                {canManage && !k.revokedAt ? (
                  <button className="btn btn-ghost" type="button" onClick={() => void onRevokeKey(k.id)}>
                    Revoke
                  </button>
                ) : null}
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  )
}
