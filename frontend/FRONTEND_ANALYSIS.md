# NRCC Frontend — Análisis Completo del Stack y Sistema de Estilos

**Fecha del Análisis**: 14 de Mayo de 2026  
**Proyecto**: Node-RED Control Center (nrcc) — Control Panel Web  
**Versión de Node**: 18+ (package.json)

---

## 📋 Resumen Ejecutivo

El frontend de NRCC es una **aplicación React 18 full-stack** que utiliza:
- **Tailwind CSS 3.4.14** como framework de styling
- **DaisyUI 5.0.0** para componentes pre-diseñados
- **Sistema de temas dual** ("corporateDark" y "corporateLight") completamente personalizado
- **Arquitectura feature-based** con thin orchestrators y hooks personalizados
- **Validación de formularios** con React Hook Form + Zod
- **Gestión de estado** con Zustand y TanStack Query

El proyecto NO usa shadcn/ui, pero implementa su propio sistema de componentes minimalista usando CVA (class-variance-authority).

---

## 1️⃣ STACK TÉCNICO ACTUAL

### Dependencies (package.json)

```json
{
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.28.0",
    "@tanstack/react-query": "^5.59.0",
    "react-hook-form": "^7.53.0",
    "zod": "^3.23.8",
    "zustand": "^5.0.0",
    "tailwindcss": "^3.4.14",
    "daisyui": "^5.0.0",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.1",
    "tailwind-merge": "^3.5.0",
    "lucide-react": "^0.460.0",
    "axios": "^1.7.7",
    "sonner": "^1.7.0",
    "date-fns": "^4.1.0"
  }
}
```

### Build Stack
- **Bundler**: Vite 7.3.3 (dev/build)
- **Testing**: Vitest 4.1.5 + React Testing Library
- **Linting**: ESLint 9.13.0 + TypeScript ESLint
- **CSS Processing**: PostCSS 8.4.47 + Autoprefixer

### Arquitectura
- **Routing**: React Router v6 (nested routes, protected routes)
- **State Management**: 
  - Zustand para global UI state
  - TanStack Query (v5) para server state / API caching
  - React Hook Form para form-local state
- **Type Safety**: TypeScript ~5.6.2 (strict mode)
- **Code Organization**: Feature-based (src/features/{name}/)

---

## 2️⃣ SISTEMA DE ESTILOS ACTUAL

### Configuración de Tailwind (tailwind.config.js)

```javascript
import daisyui from 'daisyui'

export default {
  darkMode: ['class', '[data-theme]'],
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      boxShadow: {
        'glow': '0 24px 60px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(148, 163, 184, 0.08)',
        'glow-light': '0 24px 48px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(15, 23, 42, 0.04)',
      },
    },
  },
  plugins: [daisyui],
  daisyui: {
    themes: [
      {
        corporateDark: { /* see colors section */ },
        corporateLight: { /* see colors section */ }
      }
    ]
  }
}
```

**Dark Mode Strategy**: 
- Método dual: CSS class + data-theme attribute
- DaisyUI maneja el theme switching automáticamente
- Fallback a `[data-theme]` para máxima compatibilidad

### PostCSS (postcss.config.js)
```javascript
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

---

## 3️⃣ PALETA DE COLORES

### Modo Oscuro (`corporateDark`)

**Color Primario**: RED — Operativo, urgencia, acciones principales
```
Hex: #ff3b30
RGB: 255, 59, 48
Rol: CTA primario, botones, active states, estado de error
Descripción: Rojo "signal" brillante — diseñado para operadores que necesitan claridad en consolas oscuras
```

**Color Secundario**: TEAL/CYAN — Complemento, estado alternativo
```
Hex: #10242a
RGB: 16, 36, 42
Rol: Backgrounds secundarios, fondo de inputs, estado neutral
Descripción: Azul profundo muy oscuro — casi negro con tono frío
```

**Color de Acento**: CYAN — Información, estados positivos
```
Hex: #38bdf8 (Sky Blue 300)
RGB: 56, 189, 248
Rol: Hover states, información, links
Descripción: Cyan brillante para máxima legibilidad sobre dark
```

**Base Colors (Dark)**:
| Token | Hex | Rol |
|-------|-----|-----|
| `base-100` | `#06080a` | Fondo principal (casi negro puro) |
| `base-200` | `#0c1116` | Fondo secundario (ligeramente más claro) |
| `base-300` | `#151c24` | Fondo terciario (para inputs/cards) |
| `base-content` | `#eef4f8` | Texto principal (gris muy claro) |

**Semantic Colors (Dark)**:
```
success:  #2dd4bf (Teal 400) — Operación exitosa, confirmación
warning:  #fbbf24 (Amber 300) — Precaución, requiere atención
error:    #ff3b30 (Red 500) — Crítico, destructivo
info:     #38bdf8 (Sky Blue 300) — Informativo, neutral
```

**Fondo Decorativo (Dark)**:
```
Grid: linear-gradient(90deg, rgba(56, 189, 248, 0.04) 1px, transparent 1px)
Radios: 
  - Circle at 8%, 0% → Red glow (rgba(255, 59, 48, 0.12))
  - Circle at 92%, 12% → Teal glow (rgba(45, 212, 191, 0.08))
Base: Linear gradient #0a0d10 → #06080a → #040607
```

---

### Modo Claro (`corporateLight`)

**Color Primario**: DARK RED
```
Hex: #d92d20
RGB: 217, 45, 32
Rol: CTA primario, botones, acciones
Descripción: Rojo corporate más suave que dark mode; menos agresivo para interfaces claras
```

**Color de Acento**: COBALT BLUE
```
Hex: #0369a1 (Sky Blue 700)
RGB: 3, 105, 161
Rol: Información, hover states, links
Descripción: Azul intenso con legibilidad óptima en fondo claro
```

**Base Colors (Light)**:
| Token | Hex | Rol |
|-------|-----|-----|
| `base-100` | `#f7f9fb` | Fondo principal (casi blanco, ligeramente azulado) |
| `base-200` | `#edf2f6` | Fondo secundario (gris muy claro) |
| `base-300` | `#dce5ec` | Fondo terciario (gris claro para inputs) |
| `base-content` | `#111827` | Texto principal (gris casi negro) |

**Semantic Colors (Light)**:
```
success:  #0f766e (Teal 800) — Verde corporate profesional
warning:  #b45309 (Amber 700) — Amarillo corporate
error:    #d92d20 (Red 700) — Rojo corporate
info:     #0284c7 (Sky Blue 700) — Azul corporate
```

---

### Variables CSS Definidas

El archivo `src/index.css` define 60+ custom properties por tema:

```css
[data-theme="corporateDark"] {
  /* Colors */
  --color-base-100:          #06080a;
  --color-base-200:          #0c1116;
  --color-base-300:          #151c24;
  --color-base-content:      #eef4f8;
  --color-primary:           #ff3b30;
  --color-primary-content:   #ffffff;
  --color-secondary:         #10242a;
  --color-secondary-content: #dff7f3;
  --color-accent:            #38bdf8;
  --color-accent-content:    #041016;
  --color-neutral:           #1a242d;
  --color-neutral-content:   #eef4f8;
  --color-info:              #38bdf8;
  --color-success:           #2dd4bf;
  --color-warning:           #fbbf24;
  --color-error:             #ff3b30;
  
  /* Radius & Borders */
  --radius-selector: .375rem;  /* 6px */
  --radius-field:    .25rem;   /* 4px */
  --radius-box:      .5rem;    /* 8px */
  --size-selector:   .25rem;   /* 4px */
  --size-field:      .25rem;   /* 4px */
  --border:          1px;
  --depth:           1;
  --noise:           0;
}
```

---

## 4️⃣ TIPOGRAFÍA

### Estrategia de Fuentes

**Hallazgo Importante**: El proyecto **NO importa custom web fonts**. Utiliza **system fonts** (defaults del navegador).

- **Headlines**: Usa `font-weight: 600-700` sin font-family explícito
- **Body**: Usa `font-weight: 400-500` sin font-family explícito
- **Monospace**: Lucide React para iconos; sin monospace explícito para code blocks

### Escala de Tipografía (utilities en src/index.css)

```css
.text-display {
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.1;          /* Muy apretado */
}

.text-headline {
  font-weight: 600;
  letter-spacing: -0.01em;
  line-height: 1.25;
  margin-top: 3rem;
}

.text-title {
  font-weight: 500;
  line-height: 1.3;
}

.text-body-secondary {
  line-height: 1.6;
  [dark] color: #9aa8b5;      /* Gris azulado 60% opacity */
  [light] color: rgba(17, 24, 39, 0.62);
}

.text-label {
  font-weight: 500;
  font-size: 0.75rem;         /* 12px */
  letter-spacing: 0.05em;     /* +5% */
  text-transform: uppercase;
  line-height: 1;
}
```

### Tamaños de Botón (CVA - Button.tsx)

```typescript
size: {
  sm: 'px-3 py-1.5 text-sm gap-1.5',   // 12px font
  md: 'px-4 py-2 text-sm gap-2',       // 12px font (default)
  lg: 'px-6 py-3 text-base gap-2.5',   // 14px font
}
```

### Observación
**No hay custom font imports** — esto podría ser un problema de marca si el proyecto requiere identidad visual fuerte. Recomendación: Considerar agregar **Inter**, **Geist**, o **Manrope** para diferenciación.

---

## 5️⃣ COMPONENTES UI EXISTENTES

### 5.1 Core Components (src/shared/components/ui/)

#### Button.tsx
```typescript
const buttonVariants = cva(
  'inline-flex items-center justify-center font-semibold transition-all duration-200 ...',
  {
    variants: {
      variant: {
        primary:   'btn-primary',      // Red gradient solid
        secondary: 'btn-secondary',    // Ghost border (sky blue hover)
        tertiary:  'btn-tertiary',     // Text accent (cyan)
      },
      size: {
        sm: 'px-3 py-1.5 text-sm gap-1.5',
        md: 'px-4 py-2 text-sm gap-2',
        lg: 'px-6 py-3 text-base gap-2.5',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  }
);
```

**Estilos**:
- Primary (dark): `linear-gradient(180deg, #ff4b40, #d92d20)` con glow
- Primary (light): `linear-gradient(180deg, #e5483d, #b42318)` suavizado
- Secondary: Border ghost con hover sky-blue
- Tertiary: Text-only accent cyan

---

#### Input.tsx
```typescript
export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, error, label, id, ...props }, ref) => {
    return (
      <div className="space-y-1.5">
        {label && <label className="text-label text-base-content">{label}</label>}
        <input
          className={cn(
            'flex w-full rounded-md px-3 py-2 text-sm',
            'bg-base-300 text-base-content placeholder:text-base-content/50',
            'border-none outline-none',
            'focus:ring-1 focus:ring-primary input-focus-glow',
            error && 'ring-1 ring-error input-error-glow',
          )}
          {...props}
        />
      </div>
    );
  }
);
```

**Características**:
- Label integrado (opcional)
- Focus ring color = primary
- Error ring red con glow effect
- Placeholder 50% opacity
- Rounded md (6px)

---

#### WarningBanner.tsx
```typescript
export function WarningBanner({ message, className }: WarningBannerProps) {
  return (
    <div className="flex items-center justify-between gap-2 rounded-2xl 
                    border border-warning/20 bg-warning/10 px-4 py-3 
                    text-warning-content">
      <div className="flex items-center gap-2">
        <AlertTriangle className="h-4 w-4 text-warning" />
        <span className="text-sm text-base-content">{message}</span>
      </div>
      <button className="btn btn-ghost btn-sm">
        <X className="h-3 w-3" /> Dismiss
      </button>
    </div>
  );
}
```

**Estilo**: Alert con ícono, borde amber 20% opacity, fondo amber 10% opacity

---

#### ToastViewport.tsx
```typescript
import { Toaster } from 'sonner';
```

Simple wrapper sobre **Sonner** para notificaciones toast (Tostadas).

---

### 5.2 DaisyUI Components Utilizados

El proyecto usa extensivamente componentes DaisyUI natively:

| Componente | Rol | Temas aplicados |
|-----------|-----|-----------------|
| `.btn` | Botones | btn-primary, btn-secondary, btn-danger, etc. |
| `.btn-group` | Botones agrupados | Sobreescritura de hover |
| `.menu` | Navegación sidebar | Color 72% opacity, active blue glow |
| `.card` | Tarjetas | glass-panel / surface-card styling |
| `.input` | Inputs | bg-base-300, focus glow |
| `.label` | Labels | text-label styling |
| `.badge` | Badges | Colores semánticos (success/error/warning) |
| `.divider` | Divisores | Border color 14% opacity dark |
| `.modal` | Modales | Modal overlay + modal-inner themed |
| `.table` | Tablas | table-row-hover, table-header-subtle |
| `.toggle` | Switches | Border azul 34%, checked red |
| `.select` | Dropdowns | Inherita estilos Input |
| `.skeleton` | Loading | skeleton-dark class (10% opacity) |

---

### 5.3 Custom Component Classes (src/index.css)

#### Semantic Surface Classes
```css
.glass-panel {
  /* Glassmorphism effect */
  background: linear-gradient(180deg, rgba(21, 28, 36, 0.78), rgba(12, 17, 22, 0.68));
  border-color: rgba(148, 163, 184, 0.16);
  backdrop-filter: blur(18px) saturate(140%);
}

.surface-panel {
  /* Filled panel for contained content */
  @apply rounded-lg shadow-glow;
  background: linear-gradient(180deg, rgba(21, 28, 36, 0.92), rgba(12, 17, 22, 0.9));
  border: 1px solid rgba(148, 163, 184, 0.14);
}

.surface-card {
  /* Lighter than surface-panel; more prominent */
  @apply rounded-lg shadow-glow;
  background: linear-gradient(180deg, rgba(21, 28, 36, 0.78), rgba(9, 13, 17, 0.78));
  border: 1px solid rgba(148, 163, 184, 0.13);
}
```

#### Action Button Classes
```css
.action-btn-primary   /* glass-panel + primary red border + red hover 14% opacity */
.action-btn-secondary /* glass-panel + neutral border + cyan hover 8% opacity */
.action-btn-danger    /* glass-panel + red border + red hover 12% opacity */
.action-btn-ghost     /* glass-panel + transparent border + neutral hover 8% opacity */
```

#### Modal & Overlay
```css
.modal-overlay        /* backdrop blur dark/light adaptive, z-50 */
.modal-inner          /* Inner panel with theme-aware background */
```

---

## 6️⃣ IDENTIDAD VISUAL DEL PROYECTO

### Propósito del Proyecto
Node-RED Control Center es una **UI de operación/administración** para Node-RED:
- Monitoreo de sistema (CPU, memory, disk, uptime)
- Configuración de Node-RED
- Gestión de flujos
- Gestión de librerías npm
- Backups/Restore
- Logs en tiempo real (SSE)
- Autenticación JWT multi-usuario

### "Feeling" Actual

**Estilo Visual**: **"Operativo corporativo oscuro"**
- **Dark-first**: Diseñado para operadores que pasan horas viendo dark screens
- **Rojo como error/acción**: Signal red (#ff3b30) para urgencia y claridad
- **Cyan como información**: Sky blue para datos positivos, hover states
- **Glassmorphism**: Blur backgrounds + semi-transparent cards = "futurista pero profesional"
- **Grid fondo**: Patrón grid sutil + radial gradient = "tech vibe"
- **High contrast**: Texto muy claro sobre fondos muy oscuros (accesibilidad)

### Sensación General
```
🎯 Audiencia: Operadores técnicos, DevOps, Node-RED administrators
🎨 Tono: Corporativo, profesional, "no-nonsense"
⚡ Energía: Eficiente, rápido, enfocado
🔴 Riesgo: Puede parecer frío, impersonal si no hay micro-interactions
```

### Inconsistencias Detectadas

1. **Sin web fonts custom**: Sistema fonts por defecto pueden no alinearse con brand
2. **Glassmorphism puede ser excesivo**: Blur + gradient + shadow = visual peso
3. **Color secundario (#10242a) no se usa mucho**: Definido en theme pero underutilizado
4. **Buttons: inconsistencia de contrast**: Primary buttons OK, pero secondary buttons podrían ser más legibles
5. **Tabla hover states**: Glow azul cyan es muy sutil (4-6% opacity) — puede pasar desapercibido
6. **Card shadows**: "glow" es muy fuerte (60px blur) — podría dominar layout en cards agrupadas

---

## 7️⃣ PROBLEMAS VISUALES & TOKENS FALTANTES

### 7.1 Inconsistencias en Sistema de Estilos

#### 1. Missing Typography Scale
```
ACTUAL: .text-display, .text-headline, .text-title, .text-body-secondary, .text-label
FALTA:  
  - .text-body (default para párrafos)
  - .text-caption (captions, hints)
  - .text-code (monospace para code blocks)
  - Size/weight combinations (e.g., .text-lg, .text-sm con weights)
```

#### 2. Spacing Inconsistency
```
ACTUAL: Tailwind default spacing scale (4px, 8px, 16px, 24px...)
FALTA:
  - --spacing-* custom properties no definidas en CSS variables
  - Layouts pueden no respetar consistent spacing grid
```

#### 3. Border Radius Inconsistency
```
DEFINIDO en CSS vars:
  --radius-selector: .375rem (6px)
  --radius-field:    .25rem (4px)
  --radius-box:      .5rem (8px)

PERO usados inconsistentemente:
  - Buttons: rounded-lg (8px Tailwind) vs --radius-box
  - Inputs: rounded-md (6px Tailwind) vs --radius-field
  - Cards: rounded-lg (8px) inconsistently applied
```

#### 4. Shadow Token Underutilization
```
DEFINIDO:
  shadow-glow: 60px blur (very heavy)
  shadow-glow-light: 48px blur

FALTA:
  shadow-sm (1-2px blur for subtle depth)
  shadow-md (4-6px blur for normal depth)
  shadow-lg (12-24px blur for modal depth)
```

#### 5. Color Opacity Inconsistency
```
EJEMPLO: Glass panel borders
  Dark: rgba(148, 163, 184, 0.16) = 16%
  Light: rgba(15, 23, 42, 0.08) = 8%

PERO: Hover backgrounds
  Dark actions: rgba(56, 189, 248, 0.08) = 8%
  Dark tables: rgba(56, 189, 248, 0.06) = 6%

⚠️ Sin regla clara: cuándo usar 6% vs 8% vs 12% opacity?
```

---

### 7.2 Missing Design Tokens

| Token Type | Missing | Impact |
|-----------|---------|--------|
| **Elevation** | No z-index scale defined | Modal/dropdown stacking unclear |
| **Animation** | No duration/timing tokens | Transitions hardcoded (200ms, 150ms) |
| **Focus Ring** | Partial (2px ring, 4px offset) | No ring-width scale |
| **Error States** | Only ring-error; no border glow variations | Accessibility unclear |
| **Disabled States** | Only opacity-50; no separate disabled color | Visual distinction weak |
| **Font Weights** | No @apply for weight classes | Font weights scattered (500, 600, 700) |
| **Density** | No compact/normal/spacious modes | Layouts feel always the same density |

---

### 7.3 CSS Custom Properties Not Leveraged

El archivo index.css define variables CSS pero **muchas NO se usan en componentes**:

```css
/* DEFINED but not used in components */
--radius-selector: .375rem;   /* Never see this used */
--size-selector:   .25rem;    /* ? */
--depth:           1;         /* ? */
--noise:           0;         /* ? */
```

Esto sugiere:
- ❌ Plan de diseño incompleto
- ❌ Tokens "futuros" que nunca se implementaron
- ❌ Legacy from design system spec que no se aplicó

---

## 8️⃣ ESTRUCTURA DE DIRECTORIOS (Frontend)

```
frontend/
├── src/
│   ├── features/                          # Feature-based modules
│   │   ├── auth/                          # Authentication views
│   │   ├── backups/                       # Backup management
│   │   ├── bootstrap/                     # Initial setup
│   │   ├── configuration/                 # Node-RED config editor
│   │   ├── dashboard/                     # Home page
│   │   ├── docker/                        # Docker status
│   │   ├── env-vars/                      # Environment variables
│   │   ├── environment/                   # (likely same as env-vars?)
│   │   ├── flows/                         # Flow viewer & export
│   │   ├── libraries/                     # npm package management
│   │   ├── logs/                          # Real-time logs (SSE)
│   │   ├── patterns/                      # ?
│   │   ├── runtime/                       # Process management
│   │   └── updates/                       # Self-update checks
│   ├── shared/
│   │   ├── components/
│   │   │   ├── ui/
│   │   │   │   ├── Button.tsx
│   │   │   │   ├── Input.tsx
│   │   │   │   ├── ToastViewport.tsx
│   │   │   │   ├── WarningBanner.tsx
│   │   │   │   └── index.ts
│   │   │   ├── layout/
│   │   │   │   ├── Layout.tsx
│   │   │   │   ├── ErrorBoundary.tsx
│   │   │   │   └── ...
│   │   │   └── ProtectedRoute.tsx
│   │   ├── lib/
│   │   │   ├── utils.ts           # cn() helper (clsx + tailwind-merge)
│   │   │   └── ...
│   │   ├── hooks/
│   │   └── services/
│   ├── App.tsx                    # Router setup
│   ├── main.tsx                   # Entry point
│   └── index.css                  # Global styles + theme vars
├── tailwind.config.js
├── postcss.config.js
├── vite.config.ts
├── vitest.config.ts
└── package.json
```

---

## 9️⃣ HALLAZGOS CLAVE

### ✅ Fortalezas

1. **Sistema de temas robusto**: Dos temas completos (dark/light) con variables CSS consistentes
2. **DaisyUI bien integrado**: Aprovecha componentes pre-built manteniendo customización
3. **Component library minimalista**: Button, Input, WarningBanner son bien construidos con CVA
4. **Feature-based architecture**: Facilita mantenimiento y escalabilidad
5. **Type safety**: TypeScript strict mode en todos los archivos
6. **Color palette semántico**: Success/warning/error/info colores bien definidos

### ⚠️ Debilidades

1. **Sin web fonts personalizadas**: Identity visual débil sin tipografía custom
2. **Tokens CSS parcialmente utilizados**: Variables definidas pero no aplicadas consistentemente
3. **Elevation/shadow confuso**: Shadow "glow" muy pesado; falta escala de sombras
4. **Glassmorphism puede ser excesivo**: Blur + gradient en todo puede saturar visualmente
5. **Spacing inconsistente**: No hay custom spacing tokens, depende de Tailwind defaults
6. **Focus/accessibility incompleta**: Focus rings bien, pero falta exploration de keyboard nav
7. **Disabled states débiles**: Solo opacity-50; podría haber color variation
8. **Animation tokens faltantes**: Duraciones hardcoded (200ms, 150ms) en múltiples lugares

### 🎯 Oportunidades

1. **Agregar web font** (Geist, Inter, Manrope) para diferenciación de marca
2. **Refactorizar shadow scale** (sm/md/lg/xl) con mejor contrast
3. **Simplificar glassmorphism** o hacerlo más selectivo (solo en hero/modals)
4. **Definir spacing system** explícitamente en CSS variables
5. **Complete typography scale** con todas las variaciones
6. **Animation tokens** para transiciones consistentes
7. **Accessibility audit** (WCAG 2.2) para keyboard navigation y screen readers

---

## 🔟 VALORES HEX EXACTOS (REFERENCIA RÁPIDA)

### Dark Theme
```
Primary:        #ff3b30 (Red 500)
Accent:         #38bdf8 (Sky Blue 300)
Success:        #2dd4bf (Teal 400)
Warning:        #fbbf24 (Amber 300)
Error:          #ff3b30 (Red 500)
Base-100:       #06080a (Almost black)
Base-200:       #0c1116 (Charcoal)
Base-300:       #151c24 (Slate)
Base-Content:   #eef4f8 (Off-white)
Secondary:      #10242a (Navy)
Neutral:        #1a242d (Slate-navy)
```

### Light Theme
```
Primary:        #d92d20 (Red 700)
Accent:         #0369a1 (Sky Blue 700)
Success:        #0f766e (Teal 800)
Warning:        #b45309 (Amber 700)
Error:          #d92d20 (Red 700)
Base-100:       #f7f9fb (Off-white)
Base-200:       #edf2f6 (Light gray)
Base-300:       #dce5ec (Lighter gray)
Base-Content:   #111827 (Charcoal)
Secondary:      #e4f4f1 (Teal tint)
Neutral:        #334155 (Slate)
```

### Shadows
```
glow:           0 24px 60px rgba(0, 0, 0, 0.42), 0 0 0 1px rgba(148, 163, 184, 0.08)
glow-light:     0 24px 48px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(15, 23, 42, 0.04)
```

---

## 🎨 RECOMENDACIONES INMEDIATAS

### Fase 1: Estabilización (Low Risk)
- [ ] Agregar `Inter` o `Geist` font-family
- [ ] Documentar y aplicar consistent shadow scale (sm/md/lg/xl)
- [ ] Definir elevation z-index scale en CSS variables
- [ ] Complete typography scale con .text-code, .text-caption

### Fase 2: Refinamiento (Medium Risk)
- [ ] Refactor glassmorphism — hacerlo más selectivo (solo hero + modals)
- [ ] Simplificar border radius — usar `rounded-lg` consistentemente
- [ ] Agregar animation tokens (@1 duration scale)
- [ ] Accessibility audit (WCAG 2.2) para keyboard navigation

### Fase 3: Diferenciación (Higher Risk)
- [ ] Proponer nuevo color palette si brand refresh needed
- [ ] Explore density modes (compact/normal/spacious)
- [ ] Custom iconography vs Lucide (¿requiere identidad visual más fuerte?)

---

**Análisis completado**: 14/05/2026 — Contactar para propuesta de diseño refinado
