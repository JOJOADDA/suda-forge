import { useEffect, useState, type FormEvent } from 'react'

export type AuthUser = { id: string; email: string; display_name: string; global_role: string; status: string }
export type AuthState = { user: AuthUser; session: { id: string; expires_at: string } }

type AuthPanelProps = {
  api: string
  onAuthenticated: (state: AuthState) => void
}

async function readError(response: Response, fallback: string) {
  try {
    const payload = await response.json() as { error?: string | { message?: string } }
    if (typeof payload.error === 'string') return payload.error
    if (payload.error?.message) return payload.error.message
  } catch {
    // The server may return an empty response for an infrastructure failure.
  }
  return fallback
}

export function AuthPanel({ api, onAuthenticated }: AuthPanelProps) {
  const [bootstrapRequired, setBootstrapRequired] = useState<boolean | null>(null)
  const [mode, setMode] = useState<'login' | 'bootstrap'>('login')
  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    fetch(`${api}/auth/status`, { credentials: 'include' })
      .then(async (response) => {
        if (!response.ok) throw new Error(await readError(response, 'تعذر الاتصال بخدمة المصادقة'))
        return response.json() as Promise<{ bootstrap_required: boolean }>
      })
      .then((payload) => {
        if (!active) return
        setBootstrapRequired(payload.bootstrap_required)
        setMode(payload.bootstrap_required ? 'bootstrap' : 'login')
      })
      .catch((err: unknown) => {
        if (active) setError(err instanceof Error ? err.message : 'تعذر الاتصال بخدمة المصادقة')
      })
    return () => { active = false }
  }, [api])

  async function submit(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError('')
    const endpoint = mode === 'bootstrap' ? '/auth/bootstrap' : '/auth/login'
    const body = mode === 'bootstrap' ? { email, display_name: displayName, password } : { email, password }
    try {
      const response = await fetch(`${api}${endpoint}`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!response.ok) throw new Error(await readError(response, mode === 'bootstrap' ? 'تعذر إنشاء المدير الأول' : 'بيانات الدخول غير صحيحة'))
      onAuthenticated(await response.json() as AuthState)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'تعذر إكمال العملية')
    } finally {
      setBusy(false)
    }
  }

  const checking = bootstrapRequired === null && !error
  return <main className="auth-shell" dir="rtl">
    <section className="auth-card">
      <div className="auth-brand"><span className="eyebrow">SUDA TECHNOLOGIES</span><strong>SUDA FORGE</strong></div>
      <div className="auth-heading"><span className="eyebrow">CONTROL PLANE ACCESS</span><h1>{mode === 'bootstrap' ? 'أنشئ حساب المدير الأول' : 'تسجيل الدخول'}</h1><p>{mode === 'bootstrap' ? 'أنشئ حساب Administrator الأول لبدء حماية لوحة التحكم.' : 'سجّل الدخول للوصول إلى مشاريعك وبيئات التشغيل والوكلاء.'}</p></div>
      {checking && <div className="auth-loading">جاري فحص حالة المصادقة…</div>}
      {error && <div className="error auth-error">{error}</div>}
      {!checking && <form className="auth-form" onSubmit={submit}>
        <label>البريد الإلكتروني<input type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" required /></label>
        {mode === 'bootstrap' && <label>الاسم المعروض<input value={displayName} onChange={(event) => setDisplayName(event.target.value)} autoComplete="name" required /></label>}
        <label>كلمة المرور<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete={mode === 'bootstrap' ? 'new-password' : 'current-password'} minLength={12} required /></label>
        <button type="submit" disabled={busy}>{busy ? 'جاري التحقق…' : mode === 'bootstrap' ? 'إنشاء Administrator' : 'دخول إلى SUDA FORGE'}</button>
      </form>}
      {bootstrapRequired === false && <button className="auth-switch" onClick={() => { setMode(mode === 'login' ? 'bootstrap' : 'login'); setError('') }}>{mode === 'login' ? 'إنشاء حساب المدير الأول' : 'العودة إلى تسجيل الدخول'}</button>}
    </section>
  </main>
}
