import { useEffect, useState } from 'react'

type ProjectMember = {
  project_id: string
  user_id: string
  email: string
  display_name: string
  role: 'owner' | 'editor' | 'runner' | 'viewer'
  status: string
  created_at: string
  updated_at: string
}

type Props = {
  api: string
  projectId: string
  currentUserId?: string
  globalRole?: string
}

const roles: ProjectMember['role'][] = ['owner', 'editor', 'runner', 'viewer']

export function ProjectMembersPanel({ api, projectId, currentUserId, globalRole }: Props) {
  const [members, setMembers] = useState<ProjectMember[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function loadMembers() {
    if (!projectId) return
    setError('')
    const response = await fetch(`${api}/api/projects/${projectId}/members`, { credentials: 'include' })
    if (!response.ok) throw new Error('تعذر تحميل أعضاء المشروع')
    setMembers(await response.json() as ProjectMember[])
  }

  useEffect(() => {
    loadMembers().catch((err) => setError(err instanceof Error ? err.message : 'تعذر تحميل أعضاء المشروع'))
  }, [projectId])

  async function updateRole(userId: string, role: ProjectMember['role']) {
    setBusy(true)
    setError('')
    try {
      const response = await fetch(`${api}/api/projects/${projectId}/members/${userId}`, {
        method: 'PUT', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ role }),
      })
      if (!response.ok) throw new Error((await response.json()).error?.message ?? 'تعذر تحديث الدور')
      await loadMembers()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'تعذر تحديث الدور')
    } finally {
      setBusy(false)
    }
  }

  async function removeMember(userId: string) {
    if (!window.confirm('هل تريد إزالة هذا العضو من المشروع؟')) return
    setBusy(true)
    setError('')
    try {
      const response = await fetch(`${api}/api/projects/${projectId}/members/${userId}`, { method: 'DELETE', credentials: 'include' })
      if (!response.ok) throw new Error((await response.json()).error?.message ?? 'تعذر إزالة العضو')
      await loadMembers()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'تعذر إزالة العضو')
    } finally {
      setBusy(false)
    }
  }

  const canManage = globalRole === 'admin' || globalRole === 'operator' || members.some((member) => member.user_id === currentUserId && member.role === 'owner')

  return <section className="workspace project-members">
    <div className="section-title"><div><span className="eyebrow">ACCESS CONTROL / PROJECT MEMBERS</span><h3>أعضاء المشروع</h3></div><span>{members.length} عضو</span></div>
    {!projectId ? <div className="empty"><strong>اختر مشروعًا</strong><span>ستظهر العضويات عند اختيار مشروع.</span></div> : error ? <div className="error">{error}</div> : members.length === 0 ? <div className="empty"><strong>لا يوجد أعضاء</strong><span>أنشئ المشروع أو تحقق من migration العضويات.</span></div> : <div className="member-list">{members.map((member) => <article className="member-row" key={member.user_id}><div><strong>{member.display_name || member.email}</strong><span className="muted">{member.email} · {member.status}</span></div><div className="member-actions"><select value={member.role} disabled={!canManage || busy} onChange={(event) => updateRole(member.user_id, event.target.value as ProjectMember['role'])}>{roles.map((role) => <option key={role} value={role}>{role}</option>)}</select>{canManage && <button className="ghost" disabled={busy} onClick={() => removeMember(member.user_id)}>إزالة</button>}</div></article>)}</div>}
    {!canManage && projectId && <p className="muted">صلاحيتك للعرض فقط؛ تغيير الأدوار والإزالة متاحان للمالك أو للمشرف العام.</p>}
  </section>
}
