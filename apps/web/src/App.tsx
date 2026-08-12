import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'

type Project = { id: string; name: string; slug: string; status: string; runtime_id?: string }
type Health = { application: string; runtime: string; reason?: string }
const API = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export default function App() {
  const [projects, setProjects] = useState<Project[]>([])
  const [name, setName] = useState('hello-world')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [health, setHealth] = useState<Health | null>(null)
  async function loadProjects() { const response = await fetch(`${API}/api/v1/projects`); if (!response.ok) throw new Error('تعذر تحميل المشاريع'); setProjects(await response.json()) }
  useEffect(() => { loadProjects().catch((err) => setError(err.message)); fetch(`${API}/health`).then((response) => response.json()).then(setHealth).catch(() => setHealth({ application: 'UNKNOWN', runtime: 'UNKNOWN' })) }, [])
  async function createProject(event: FormEvent) { event.preventDefault(); setLoading(true); setError(''); try { const response = await fetch(`${API}/api/v1/projects`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); if (!response.ok) throw new Error((await response.json()).error?.message ?? 'فشل إنشاء المشروع'); setName(''); await loadProjects() } catch (err) { setError(err instanceof Error ? err.message : 'حدث خطأ غير معروف') } finally { setLoading(false) } }
  async function changeStatus(project: Project, action: 'start' | 'stop') { setError(''); try { const response = await fetch(`${API}/api/v1/projects/${project.id}/${action}`, { method: 'POST' }); if (!response.ok) throw new Error((await response.json()).error?.message ?? 'فشل تغيير الحالة'); await loadProjects() } catch (err) { setError(err instanceof Error ? err.message : 'حدث خطأ غير معروف') } }
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
  </main>
}
