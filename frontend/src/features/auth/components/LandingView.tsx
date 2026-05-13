import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../../../shared/hooks/useTheme';
import { authService } from '../services/authService';
import {
  GitBranch,
  Activity,
  Container,
  Archive,
  Settings2,
  Users,
  Sun,
  Moon,
  Monitor,
  type LucideIcon,
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface Feature {
  title: string;
  description: string;
  icon: LucideIcon;
}

// ============================================================================
// Feature Data
// ============================================================================

const features: Feature[] = [
  {
    title: 'Flows Management',
    description: 'Deploy and manage Node-RED flows',
    icon: GitBranch,
  },
  {
    title: 'Real-time Monitoring',
    description: 'Live status and metrics',
    icon: Activity,
  },
  {
    title: 'Docker Management',
    description: 'Control Node-RED container',
    icon: Container,
  },
  {
    title: 'Backups',
    description: 'Backup and restore configuration',
    icon: Archive,
  },
  {
    title: 'Environment Variables',
    description: 'Secure env var management',
    icon: Settings2,
  },
  {
    title: 'User Management',
    description: 'Multi-user access control',
    icon: Users,
  },
];

// ============================================================================
// Hero Component
// ============================================================================

function Hero() {
  const navigate = useNavigate();

  const scrollToFeatures = () => {
    document.getElementById('features')?.scrollIntoView({ behavior: 'smooth' });
  };

  return (
    <section className="relative overflow-hidden py-20 lg:py-32">
      <div className="pointer-events-none absolute inset-0 bg-hero-grid" />
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="relative z-10 text-center max-w-4xl mx-auto">
          <p className="mb-5 text-xs uppercase tracking-[0.32em] text-base-content/55">The orchestrator's console</p>
          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-base-content">
            Node-RED <span className="text-primary">Control Center</span>
          </h1>
          <p className="mt-6 text-xl text-base-content/70 max-w-2xl mx-auto">
            Centralized management for your Node-RED instances. Deploy flows, monitor
            performance, and manage configurations from a single interface.
          </p>
          <div className="mt-10 flex flex-col sm:flex-row gap-4 justify-center">
            <button
              onClick={() => navigate('/login')}
              className="btn btn-primary px-8"
            >
              Iniciar Sesión
            </button>
            <button
              onClick={scrollToFeatures}
              className="btn btn-secondary px-8"
            >
              Ver Features
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}

// ============================================================================
// FeatureGrid Component
// ============================================================================

function FeatureCard({ feature }: { feature: Feature }) {
  const Icon = feature.icon;
  return (
    <div className="group surface-panel border border-border p-6 transition-transform duration-200 hover:-translate-y-1">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/12 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-content">
        <Icon className="w-6 h-6" />
      </div>
      <h3 className="mb-2 text-lg font-semibold text-base-content">{feature.title}</h3>
      <p className="text-sm text-base-content/70">{feature.description}</p>
    </div>
  );
}

function FeatureGrid() {
  return (
    <section id="features" className="py-20">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-2xl mx-auto mb-16">
          <h2 className="text-3xl font-bold text-base-content">Everything You Need</h2>
          <p className="mt-4 text-base-content/70">
            Complete toolkit for managing Node-RED at scale
          </p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 max-w-6xl mx-auto">
          {features.map((feature) => (
            <FeatureCard key={feature.title} feature={feature} />
          ))}
        </div>
      </div>
    </section>
  );
}

// ============================================================================
// LandingView Component
// ============================================================================

export function LandingView() {
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();

  // Smart redirect: check auth status and route accordingly
  useEffect(() => {
    authService.getStatus()
      .then(status => {
        if (!status.initialized) {
          navigate('/setup', { replace: true });
        } else {
          const token = authService.getToken();
          if (token) {
            authService.getMe()
              .then(() => navigate('/dashboard', { replace: true }))
              .catch(() => navigate('/login', { replace: true }));
          } else {
            navigate('/login', { replace: true });
          }
        }
      })
      .catch(() => navigate('/login', { replace: true }));
  }, [navigate]);

  const cycleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    setTheme(next);
  };

  const getThemeIcon = () => {
    if (theme === 'dark') return <Moon className="w-5 h-5" />;
    if (theme === 'light') return <Sun className="w-5 h-5" />;
    return <Monitor className="w-5 h-5" />;
  };

  const getThemeLabel = () => {
    if (theme === 'dark') return 'Modo oscuro';
    if (theme === 'light') return 'Modo claro';
    return 'Sistema';
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Header with Theme Toggle */}
      <header className="absolute top-0 left-0 right-0 z-50">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-end">
          <button
            onClick={cycleTheme}
            className="p-3 rounded-xl text-base-content hover:bg-base-300/50 transition-colors"
            title={getThemeLabel()}
          >
            {getThemeIcon()}
          </button>
        </div>
      </header>
      <Hero />
      <FeatureGrid />
    </div>
  );
}
