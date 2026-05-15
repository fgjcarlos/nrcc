import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../../../shared/hooks/useTheme';
import { authService } from '../services/authService';
import {
  Activity,
  Archive,
  ArrowRight,
  CheckCircle2,
  Container,
  GitBranch,
  Monitor,
  Moon,
  Settings2,
  ShieldCheck,
  Sparkles,
  Sun,
  type LucideIcon,
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface Capability {
  title: string;
  description: string;
  icon: LucideIcon;
  accent: string;
}

type LandingAuthState = 'checking' | 'authenticated' | 'login-required' | 'setup-required';

// ============================================================================
// Feature Data
// ============================================================================

const capabilities: Capability[] = [
  {
    title: 'Orquestación de flows',
    description:
      'Despliega, versiona y revisa flujos Node-RED con una experiencia pensada para equipos OT/IT.',
    icon: GitBranch,
    accent: 'from-primary/20 to-primary/5 text-primary',
  },
  {
    title: 'Observabilidad industrial',
    description:
      'Supervisa estado, métricas y señales operativas para detectar incidencias antes de tocar producción.',
    icon: Activity,
    accent: 'from-accent/20 to-accent/5 text-accent',
  },
  {
    title: 'Runtime bajo control',
    description:
      'Gestiona contenedores, entorno y librerías npm sin saltar entre terminales ni paneles aislados.',
    icon: Container,
    accent: 'from-info/20 to-info/5 text-info',
  },
  {
    title: 'Cambios recuperables',
    description:
      'Protege configuración, backups y variables críticas con acciones claras y trazables.',
    icon: Archive,
    accent: 'from-success/20 to-success/5 text-success',
  },
];

const trustSignals = [
  'Pensado para operadores, integradores y desarrolladores de automatización',
  'Unifica flows, Docker, librerías, backups y configuración',
  'Diseño responsive y accesible para escritorio, tablet y móvil',
];

// ============================================================================
// Components
// ============================================================================

function CapabilityCard({ capability }: { capability: Capability }) {
  const Icon = capability.icon;

  return (
    <article className="group relative overflow-hidden rounded-3xl border border-border bg-base-100/70 p-6 shadow-glow transition duration-200 hover:-translate-y-1 hover:border-primary/40">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-primary via-accent to-primary opacity-70" />
      <div
        className={`mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br ${capability.accent}`}
      >
        <Icon className="h-6 w-6" aria-hidden="true" />
      </div>
      <h3 className="text-lg font-semibold text-base-content">{capability.title}</h3>
      <p className="mt-3 text-sm leading-6 text-base-content/70">{capability.description}</p>
    </article>
  );
}

function SignalMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-border bg-base-100/60 p-4">
      <dt className="text-xs uppercase tracking-[0.2em] text-base-content/50">{label}</dt>
      <dd className="mt-2 text-2xl font-semibold text-base-content">{value}</dd>
    </div>
  );
}

// ============================================================================
// LandingView Component
// ============================================================================

export function LandingView() {
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();
  const [authState, setAuthState] = useState<LandingAuthState>('checking');

  useEffect(() => {
    let isMounted = true;

    authService
      .getStatus()
      .then(async (status) => {
        if (!isMounted) return;

        if (!status.initialized) {
          setAuthState('setup-required');
          return;
        }

        const token = authService.getToken();
        if (!token) {
          setAuthState('login-required');
          return;
        }

        try {
          await authService.getMe();
          if (isMounted) setAuthState('authenticated');
        } catch {
          if (isMounted) setAuthState('login-required');
        }
      })
      .catch(() => {
        if (isMounted) setAuthState('login-required');
      });

    return () => {
      isMounted = false;
    };
  }, []);

  const cycleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    setTheme(next);
  };

  const getThemeIcon = () => {
    if (theme === 'dark') return <Moon className="h-5 w-5" aria-hidden="true" />;
    if (theme === 'light') return <Sun className="h-5 w-5" aria-hidden="true" />;
    return <Monitor className="h-5 w-5" aria-hidden="true" />;
  };

  const getThemeLabel = () => {
    if (theme === 'dark') return 'Modo oscuro';
    if (theme === 'light') return 'Modo claro';
    return 'Sistema';
  };

  const primaryActionLabel = authState === 'authenticated' ? 'Iniciar proceso' : 'Loguearse';
  const primaryActionHelp =
    authState === 'authenticated'
      ? 'Accede al dashboard operativo para continuar con el flujo principal.'
      : 'Inicia sesión para desbloquear el flujo principal de administración.';

  const handlePrimaryAction = () => {
    if (authState === 'setup-required') {
      navigate('/setup');
      return;
    }

    if (authState === 'authenticated') {
      navigate('/dashboard');
      return;
    }

    navigate('/login');
  };

  return (
    <div className="app-shell min-h-screen overflow-hidden bg-background text-base-content">
      <header className="relative z-20 border-b border-border/70 bg-base-100/55 backdrop-blur-xl">
        <nav
          className="container mx-auto flex items-center justify-between px-4 py-4 sm:px-6 lg:px-8"
          aria-label="Navegación principal"
        >
          <div className="flex items-center gap-3">
            <div className="sidebar-brand-mark flex h-11 w-11 items-center justify-center rounded-2xl border">
              <Sparkles className="h-5 w-5 text-primary" aria-hidden="true" />
            </div>
            <div>
              <p className="text-sm font-semibold text-base-content">NRCC</p>
              <p className="text-xs uppercase tracking-[0.2em] text-base-content/50">Control Center</p>
            </div>
          </div>

          <button
            type="button"
            onClick={cycleTheme}
            className="theme-toggle-shell inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm text-base-content transition-colors hover:text-primary"
            aria-label={`Cambiar tema: ${getThemeLabel()}`}
            title={getThemeLabel()}
          >
            {getThemeIcon()}
            <span className="hidden sm:inline">{getThemeLabel()}</span>
          </button>
        </nav>
      </header>

      <main>
        <section className="relative px-4 py-14 sm:px-6 sm:py-20 lg:px-8 lg:py-24">
          <div className="pointer-events-none absolute inset-0 bg-hero-grid opacity-50" />
          <div className="pointer-events-none absolute left-1/2 top-20 h-72 w-72 -translate-x-1/2 rounded-full bg-primary/10 blur-3xl" />

          <div className="container relative z-10 mx-auto grid max-w-7xl items-center gap-10 lg:grid-cols-[1.08fr_0.92fr]">
            <div>
              <p className="mb-5 inline-flex items-center gap-2 rounded-full border border-primary/25 bg-primary/10 px-4 py-2 text-xs font-semibold uppercase tracking-[0.22em] text-primary">
                <ShieldCheck className="h-4 w-4" aria-hidden="true" />
                Consola OT/IT para Node-RED
              </p>

              <h1 className="max-w-4xl text-4xl font-black tracking-tight text-base-content sm:text-5xl lg:text-7xl">
                Node-RED <span className="text-primary">Control Center</span>
              </h1>

              <p className="mt-6 max-w-2xl text-lg leading-8 text-base-content/75 sm:text-xl">
                Controla instancias Node-RED desde una consola operativa que reúne flows,
                runtime, Docker, librerías, backups y configuración crítica para entornos de automatización.
              </p>

              <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
                <button
                  type="button"
                  onClick={handlePrimaryAction}
                  disabled={authState === 'checking'}
                  className="inline-flex items-center justify-center gap-2 rounded-2xl bg-primary px-6 py-3 font-semibold text-primary-content shadow-glow transition hover:bg-primary/90 disabled:cursor-wait disabled:opacity-70"
                  aria-describedby="landing-primary-action-help"
                >
                  {authState === 'checking'
                    ? 'Comprobando acceso…'
                    : authState === 'setup-required'
                      ? 'Crear administrador'
                      : primaryActionLabel}
                  <ArrowRight className="h-5 w-5" aria-hidden="true" />
                </button>
                <a
                  href="#capabilities"
                  className="inline-flex items-center justify-center rounded-2xl border border-border bg-base-100/70 px-6 py-3 font-semibold text-base-content transition hover:border-accent/50 hover:text-accent"
                >
                  Ver capacidades
                </a>
              </div>
              <p id="landing-primary-action-help" className="mt-3 text-sm text-base-content/60">
                {authState === 'setup-required'
                  ? 'El sistema necesita crear el primer usuario administrador antes de continuar.'
                  : primaryActionHelp}
              </p>

              <ul className="mt-8 grid gap-3 text-sm text-base-content/75">
                {trustSignals.map((signal) => (
                  <li key={signal} className="flex items-start gap-3">
                    <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-success" aria-hidden="true" />
                    <span>{signal}</span>
                  </li>
                ))}
              </ul>
            </div>

            <aside className="surface-panel relative overflow-hidden rounded-[2rem] border p-6 shadow-glow" aria-label="Resumen de NRCC">
              <div className="absolute right-0 top-0 h-40 w-40 translate-x-10 -translate-y-10 rounded-full bg-accent/20 blur-3xl" />
              <div className="relative">
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Estado de plataforma</p>
                    <h2 className="mt-2 text-2xl font-bold text-base-content">Preparado para operar</h2>
                  </div>
                  <div className="rounded-2xl border border-success/30 bg-success/10 p-3 text-success">
                    <Settings2 className="h-6 w-6" aria-hidden="true" />
                  </div>
                </div>

                <dl className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-3 lg:grid-cols-1 xl:grid-cols-3">
                  <SignalMetric label="Scope" value="Flows" />
                  <SignalMetric label="Runtime" value="Docker" />
                  <SignalMetric label="Recovery" value="Backups" />
                </dl>

                <div className="mt-6 rounded-3xl border border-border bg-base-100/60 p-5">
                  <p className="text-sm font-semibold text-base-content">Flujo recomendado</p>
                  <ol className="mt-4 space-y-3 text-sm text-base-content/70">
                    <li className="flex gap-3">
                      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/15 text-xs font-bold text-primary">
                        1
                      </span>
                      Revisar salud del runtime y contenedor Node-RED.
                    </li>
                    <li className="flex gap-3">
                      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-accent/15 text-xs font-bold text-accent">
                        2
                      </span>
                      Validar flows, variables y librerías antes de desplegar cambios.
                    </li>
                    <li className="flex gap-3">
                      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-success/15 text-xs font-bold text-success">
                        3
                      </span>
                      Generar backup y continuar desde el dashboard principal.
                    </li>
                  </ol>
                </div>
              </div>
            </aside>
          </div>
        </section>

        <section id="capabilities" className="px-4 pb-16 sm:px-6 lg:px-8" aria-labelledby="capabilities-heading">
          <div className="container mx-auto max-w-7xl">
            <div className="mb-8 max-w-3xl">
              <p className="text-xs font-semibold uppercase tracking-[0.24em] text-accent">Capacidades clave</p>
              <h2 id="capabilities-heading" className="mt-3 text-3xl font-bold text-base-content sm:text-4xl">
                Una entrada clara para pasar de contexto a acción
              </h2>
              <p className="mt-4 text-base leading-7 text-base-content/70">
                La home presenta qué hace NRCC, a quién ayuda y por qué usarlo antes de entrar al flujo operativo.
              </p>
            </div>

            <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-4">
              {capabilities.map((capability) => (
                <CapabilityCard key={capability.title} capability={capability} />
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}
