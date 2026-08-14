import { useEffect, useState } from 'react'

type AuditEvent = {
  id: string
  actor_user_id?: string
  action: string
  resource_type: string
  resource_id?: string
  outcome: 'success' | 'denied' | 'failure'
  metadata: Record<string, unknown>
  created_at: string
}

type Props = { api: string; projectId: string }

export function ProjectAuditPanel({ api, projectId }: Props) {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    if (!projectId) { setEvents([]); return }
    fetch(`${api}/api/projects/${projectId}/audit?limit=100`, { credentials: 'include' })
      .then(async (response) => {
        if (!response.ok) throw new Error('تعذر تحميل سجل التدقيق')
        return await response.json() as AuditEvent[]
      })
      .then((payload) => { setEvents(payload); setError('') })
      .catch((err) => setError(err instanceof Error ? err.message : 'تعذر تحميل سجل التدقيق'))
  }, [api, projectId])

  return <section className="workspace audit-panel">
    <div className="section-title"><div><span className="eyebrow">GOVERNANCE / AUDIT TRAIL</span><h3>سجل التدقيق</h3></div><span>{events.length} حدث</span></div>
    {!projectId ? <div className="empty"><strong>اختر مشروعًا</strong><span>سيظهر سجل التدقيق للمشروع المحدد.</span></div> : error ? <div className="error">{error}</div> : events.length === 0 ? <div className="empty"><strong>لا توجد أحداث تدقيق بعد.</strong><span>سيتم عرض العمليات المسجلة من الخدمات التي تكتب audit_events.</span></div> : <div className="event-stream">{events.map((event) => <div className="event" key={event.id}><span>{event.outcome} · {event.action} · {event.resource_type} · {new Date(event.created_at).toLocaleString()}</span><p>{event.resource_id ?? 'project'}{event.actor_user_id ? ` · actor ${event.actor_user_id}` : ''}</p></div>)}</div>}
  </section>
}
