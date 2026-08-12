import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'

type Project = { id: string; name: string; slug: string; status: string; runtime_id?: string }
type Health = { application: string; runtime: string; reason?: string }
type Agent = { id: string; display_name: string; adapter: string; status: string }
type AgentSession = { id: string; project_id: string; agent_id: string; status: string; runtime_id: string; working_directory: string }
type AgentEvent = { type: string; normalized?: { text?: string }; raw?: Record<string, unknown> }
type Model = { id: string; provider_id: string; display_name: string; context_window: number; local: boolean; remote: boolean; coding: boolean; reasoning: boolean; tool_use: boolean }
type RoutingDecision = { selected_model?: { provider_id: string; model_id: string }; reason?: string; alternatives?: { model: { provider_id: string; model_id: string }; score: number }[]; estimated_cost?: number; confidence?: number }
const API = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export default function App() {
  const [projects, setProjects] = useState<Project[]>([])
  const [name, setName] = useState('hello-world')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [health, setHealth] = useState<Health | null>(null)
  const [agents, setAgents] = useState<Agent[]>([])
  const [selectedProject, setSelectedProject] = useState('')
  const [selectedAgent, setSelectedAgent] = useState('codex')
  const [session, setSession] = useState<AgentSession | null>(null)
  const [agentEvents, setAgentEvents] = useState<AgentEvent[]>([])
  const [message, setMessage] = useState('')
  const [models, setModels] = useState<Model[]>([])
  const [routingPolicy, setRoutingPolicy] = useState('BALANCED')
  const [routingDecision, setRoutingDecision] = useState<RoutingDecision | null>(null)
  async function loadProjects(): Promise<Project[]> { const response = await fetch(`${API}/api/v1/projects`); if (!response.ok) throw new Error('تعذر تحميل المشاريع'); const items: Project[] = await response.json(); setProjects(items); return items }
  useEffect(() => { loadProjects().then((items) => { if (items[0]) setSelectedProject(items[0].id) }).catch((err) => setError(err.message)); fetch(`${API}/health`).then((response) => response.json()).then(setHealth).catch(() => setHealth({ application: 'UNKNOWN', runtime: 'UNKNOWN' })); fetch(`${API}/api/agents`).then((response) => response.ok ? response.json() : []).then(setAgents).catch(() => setAgents([])); fetch(`${API}/api/models`).then((response) => response.ok ? response.json() : []).then(setModels).catch(() => setModels([])) }, [])
  async function createProject(event: FormEvent) { event.preventDefault(); setLoading(true); setError(''); try { const response = await fetch(`${API}/api/v1/projects`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); if (!response.ok) throw new Error((await response.json()).error?.message ?? 'فشل إنشاء المشروع'); setName(''); await loadProjects() } catch (err) { setError(err instanceof Error ? err.message : 'حدث خطأ غير معروف') } finally { setLoading(false) } }
  async function changeStatus(project: Project, action: 'start' | 'stop') { setError(''); try { const response = await fetch(`${API}/api/v1/projects/${project.id}/${action}`, { method: 'POST' }); if (!response.ok) throw new Error((await response.json()).error?.message ?? 'فشل تغيير الحالة'); await loadProjects() } catch (err) { setError(err instanceof Error ? err.message : 'حدث خطأ غير معروف') } }
  async function createAgentSession(event: FormEvent) { event.preventDefault(); if (!selectedProject) { setError('اختر مشروعًا أولًا'); return }; setError(''); try { const response = await fetch(`${API}/api/projects/${selectedProject}/agent-sessions`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ agent_id: selectedAgent, runtime_id: `${selectedProject}_runtime`, working_directory: '/workspace' }) }); if (!response.ok) throw new Error((await response.json()).error?.message ?? 'تعذر إنشاء الجلسة'); setSession(await response.json()); setAgentEvents([]) } catch (err) { setError(err instanceof Error ? err.message : 'حدث خطأ غير معروف') } }
  async function sendAgentMessage(event: FormEvent) { event.preventDefault(); if (!session || !message.trim()) return; const response = await fetch(`${API}/api/projects/${session.project_id}/agent-sessions/${session.id}/messages`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ message }) }); if (!response.ok) setError((await response.json()).error?.message ?? 'تعذر إرسال الرسالة'); setMessage(''); const events = await fetch(`${API}/api/projects/${session.project_id}/agent-sessions/${session.id}/events`).then((r) => r.ok ? r.json() : []); setAgentEvents(events) }
  async function previewRouting(event: FormEvent) { event.preventDefault(); const response = await fetch(`${API}/api/model-routing/decide`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ project_id: selectedProject, agent_id: selectedAgent, policy: routingPolicy, privacy_policy: 'PUBLIC', local_policy: 'REMOTE_ALLOWED', available_runtime: true, task_profile: { task_type: 'REFACTOR', context_required: 1000, reasoning_required: true, privacy_classification: 'PUBLIC' } }) }); const payload = await response.json(); if (!response.ok) { setError(payload.error ?? 'لا يوجد نموذج متوافق'); setRoutingDecision(payload.decision ?? null); return }; setRoutingDecision(payload) }
  return <main dir="rtl">
    <header><div><span className="eyebrow">SUDA TECHNOLOGIES</span><h1>SUDA FORGE</h1><p>كل مشروع هو Project Computer حقيقي، وليس مجرد مجلد أو نافذة محادثة.</p></div><div className="system"><span className="dot" /> Control Plane online</div></header>
    {health?.runtime === 'BLOCKED' && <div className="runtime-warning"><strong>LXC unavailable</strong><span>{health.reason ?? 'هذا المضيف لا يحقق متطلبات Project Computer.'}</span></div>}
    <section className="hero"><div><span className="eyebrow">LOCAL-FIRST SOFTWARE FACTORY</span><h2>ابنِ داخل كمبيوتر مشروعك.</h2><p>ابدأ من بيئة Linux معزولة، ثم أضف Terminal وFiles وGit وPreview ووكلاء الذكاء الاصطناعي فوق أساس قابل للتحقق.</p></div><div className="formula">PROJECT<br /><b>+</b> RUNTIME<br /><b>+</b> VERIFICATION</div></section>
    <section className="workspace"><div className="section-title"><div><span className="eyebrow">CONTROL PLANE / PROJECTS</span><h3>مشاريعك</h3></div><span>{projects.length} مشروع</span></div>
      <form onSubmit={createProject} className="create"><input value={name} onChange={(e) => setName(e.target.value)} placeholder="اسم المشروع" required /><button disabled={loading}>{loading ? 'جارٍ الإنشاء…' : 'إنشاء Project Computer'}</button></form>
      {error && <div className="error">{error}</div>}
      {health && <div className="health-line"><span>Application: {health.application}</span><span>Runtime host: {health.runtime}</span></div>}
      <div className="grid">{projects.map((project) => <article className="card" key={project.id}><div className="card-top"><span className={`status status-${project.status.toLowerCase()}`} />{project.status}</div><h4>{project.name}</h4><p className="muted">/{project.slug}</p><div className="capabilities"><span>Linux</span><span>Filesystem</span><span>Git</span><span>PTY</span></div><div className="actions">{project.status === 'READY' || project.status === 'STOPPED' ? <button onClick={() => changeStatus(project, 'start')}>Start</button> : null}{project.status === 'RUNNING' ? <button onClick={() => changeStatus(project, 'stop')}>Stop</button> : null}<button className="ghost">Open Computer</button></div></article>)}</div>
      {projects.length === 0 && <div className="empty"><strong>لا توجد مشاريع بعد.</strong><span>أنشئ hello-world لبدء أول vertical slice حقيقي.</span></div>}
    </section>
    <section className="workspace agent-surface"><div className="section-title"><div><span className="eyebrow">AGENT FABRIC / NORMALIZED EVENTS</span><h3>Agent Session</h3></div><span>{session?.status ?? 'لا توجد جلسة'}</span></div>
      <form onSubmit={createAgentSession} className="agent-controls"><select value={selectedProject} onChange={(e) => setSelectedProject(e.target.value)}><option value="">اختر المشروع</option>{projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}</select><select value={selectedAgent} onChange={(e) => setSelectedAgent(e.target.value)}>{agents.length ? agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.display_name}</option>) : <option value="codex">Codex</option>}</select><button disabled={!selectedProject}>Create session</button></form>
      {session && <form onSubmit={sendAgentMessage} className="create"><input value={message} onChange={(e) => setMessage(e.target.value)} placeholder="Send a provider-neutral agent message" /><button disabled={!message.trim()}>Send</button></form>}
      <div className="event-stream">{agentEvents.length ? agentEvents.map((event, index) => <div className="event" key={`${event.type}-${index}`}><span>{event.type}</span><p>{event.normalized?.text ?? JSON.stringify(event.normalized ?? event.raw ?? {})}</p></div>) : <div className="empty"><strong>لا توجد أحداث بعد.</strong><span>الأحداث المعروضة هنا normalized ولا تقرأ مخرجات CLI مباشرة.</span></div>}</div>
    </section>
    <section className="workspace model-center"><div className="section-title"><div><span className="eyebrow">MODEL FABRIC / DETERMINISTIC ROUTING</span><h3>Model Center</h3></div><span>{models.length} models</span></div>
      <div className="model-grid">{models.map((model) => <article className="model-card" key={model.id}><div><strong>{model.display_name}</strong><span>{model.provider_id} · {model.local ? 'LOCAL' : 'REMOTE'}</span></div><p>{model.context_window.toLocaleString()} context · {model.coding ? 'coding' : 'general'} · {model.tool_use ? 'tools' : 'no tools'}</p></article>)}</div>
      <form onSubmit={previewRouting} className="agent-controls"><select value={routingPolicy} onChange={(e) => setRoutingPolicy(e.target.value)}><option>BALANCED</option><option>BEST</option><option>CHEAPEST</option><option>FASTEST</option><option>LOCAL_FIRST</option></select><button disabled={!models.length}>Preview routing</button></form>
      {routingDecision && <div className="routing-result"><strong>Selected: {routingDecision.selected_model?.model_id ?? 'BLOCKED'}</strong><span>{routingDecision.reason}</span><small>Alternatives: {routingDecision.alternatives?.map((item) => item.model.model_id).join(', ') || 'none'} · estimated cost: {routingDecision.estimated_cost ?? 0}</small></div>}
    </section>
  </main>
}
