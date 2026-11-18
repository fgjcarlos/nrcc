import { NrccMark } from '@/shared/components/NrccMark';
import { ThemeToggle } from '@/shared/components/ThemeToggle';
import { LocaleSwitcher } from '@/i18n';
import { sectionId } from './lib/tokens';
import { ColourSection, LogoSection, MotionSection, SpacingSection, TypeSection, VoiceSection } from './Foundations';
import {
  ButtonsSection,
  CircuitSection,
  DialogsSection,
  FieldsSection,
  StatusSection,
  TablesSection,
  TabsSection,
} from './Components';

const INDEX = [
  { group: 'Foundations', sections: ['Colour', 'Type', 'Spacing, radii and elevation', 'Motion', 'Voice', 'Logo'] },
  { group: 'Components', sections: ['Buttons', 'Fields', 'Dialogs', 'Tabs', 'Tables', 'Status and feedback', 'Change circuit'] },
];

/**
 * The NRCC brand book (S5): a dev-only route (App.tsx registers it only
 * when import.meta.env.DEV) that renders the real tokens and the real
 * shared components, so it cannot drift from the app.
 */
export function StyleguideView() {
  return (
    <div className="min-h-dvh bg-ds-bg-void text-ds-text-primary">
      <header className="sticky top-0 z-20 flex flex-wrap items-center gap-3 border-b border-ds-border-default bg-ds-bg-surface px-4 py-2 sm:px-6">
        <NrccMark size="sm" />
        <div className="mr-auto min-w-0">
          <h1 className="text-xl font-bold">NRCC styleguide</h1>
          <p className="text-xs text-ds-text-secondary">Dev only: absent from production builds. Every value is read from the running CSS.</p>
        </div>
        <ThemeToggle />
        <LocaleSwitcher />
      </header>
      <div className="mx-auto grid max-w-screen-2xl gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[12rem_minmax(0,1fr)] lg:gap-10">
        <nav aria-label="Styleguide sections" className="lg:sticky lg:top-24 lg:self-start">
          <div className="flex flex-wrap gap-x-8 gap-y-4 lg:flex-col">
            {INDEX.map(({ group, sections }) => (
              <div key={group}>
                <p className="text-xs font-semibold text-ds-text-secondary">{group}</p>
                <ul className="mt-1 flex flex-wrap gap-x-3 lg:flex-col">
                  {sections.map((title) => (
                    <li key={title}>
                      <a href={`#${sectionId(title)}`} className="inline-block py-1 text-sm text-ds-text-primary hover:text-ds-on-tint-primary hover:underline">
                        {title}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </nav>
        <main className="min-w-0 space-y-10">
          <ColourSection />
          <TypeSection />
          <SpacingSection />
          <MotionSection />
          <VoiceSection />
          <LogoSection />
          <ButtonsSection />
          <FieldsSection />
          <DialogsSection />
          <TabsSection />
          <TablesSection />
          <StatusSection />
          <CircuitSection />
        </main>
      </div>
    </div>
  );
}
