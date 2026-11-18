import { useState } from 'react';
import { Inbox, ListTree, FileText } from 'lucide-react';
import { useT } from '@/i18n';
import { Button } from '@/shared/components/ui/Button';
import { Input } from '@/shared/components/ui/Input';
import { Select } from '@/shared/components/ui/Select';
import { Textarea } from '@/shared/components/ui/Textarea';
import { Toggle } from '@/shared/components/ui/Toggle';
import { Dialog, DialogBody, DialogFooter } from '@/shared/components/ui/Dialog';
import { Tabs, TabPanel } from '@/shared/components/ui/Tabs';
import { Table } from '@/shared/components/ui/Table';
import { StatusChip } from '@/shared/components/ui/StatusChip';
import { EmptyState, ErrorState, Skeleton } from '@/shared/components/feedback';
import { StateContainer } from '@/shared/components/StateContainer';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { ChangeCircuit } from '@/features/configuration/components/ChangeCircuit';
import { deriveCircuit, type CircuitInput } from '@/features/configuration/lib/changeCircuit';
import { Section, Specimen, Tag } from './parts';

const noop = () => {};

export function ButtonsSection() {
  const { t } = useT();
  const variants = [
    { variant: 'primary', label: t('configuration:circuit.review', { count: 3 }), use: 'Action: sheet ground, conductor-blue edge and label. "Review N changes" under the circuit is the canonical instance.' },
    { variant: 'solid', label: t('common:saveChanges'), use: 'Solid primary: conductor-blue fill. The affirmative action in a dialog footer.' },
    { variant: 'secondary', label: t('configuration:advanced.unlockToEdit'), use: 'Secondary: control-border edge, ink label. "Unlock to edit settings.js" is the canonical instance.' },
    { variant: 'ghost', label: t('configuration:discardChanges'), use: 'Ghost: transparent edge. "Discard changes" under the circuit.' },
    { variant: 'danger', label: t('common:delete'), use: 'Danger: critical-red edge and label, destructive actions only (The Red Is Rare Rule).' },
  ] as const;
  return (
    <Section title="Buttons" intro="Flat sheet, conductor-coloured edge; the colour is the job. 4px corners, 36px default height; sm and lg only change the target. Disabled drops to 50% opacity.">
      {variants.map(({ variant, label, use }) => (
        <div key={variant} className="space-y-2">
          <p className="text-sm text-ds-text-secondary">
            <Tag>{variant}</Tag> · {use}
          </p>
          <div className="flex flex-wrap items-end gap-4 rounded-box border border-ds-border-default bg-ds-bg-surface p-4">
            {(['sm', 'md', 'lg'] as const).map((size) => (
              <div key={size} className="space-y-1">
                <Tag>{size}</Tag>
                <Button type="button" variant={variant} size={size} className="block">
                  {label}
                </Button>
              </div>
            ))}
            <div className="space-y-1">
              <Tag>disabled</Tag>
              <Button type="button" variant={variant} disabled className="block">
                {label}
              </Button>
            </div>
            <div className="space-y-1">
              <Tag>focus (forced)</Tag>
              <Button type="button" variant={variant} className="block outline-2 outline-offset-2 outline-ds-accent-primary">
                {label}
              </Button>
            </div>
          </div>
        </div>
      ))}
    </Section>
  );
}

export function FieldsSection() {
  const { t } = useT();
  const [toggle, setToggle] = useState(true);
  const types = (
    <>
      <option>{t('env-vars:typeString')}</option>
      <option>{t('env-vars:typeNumber')}</option>
      <option>{t('env-vars:typeBoolean')}</option>
      <option>{t('env-vars:typeSecret')}</option>
    </>
  );
  const na = (reason: string) => <p className="text-xs text-ds-text-muted">{reason}</p>;
  const states = ['rest', 'invalid', 'disabled', 'read-only'] as const;
  const columns = [
    {
      name: 'Input',
      render: (state: (typeof states)[number]) => (
        <Input
          id={`sg-input-${state}`}
          label={t('env-vars:modalKeyLabel')}
          defaultValue={state === 'invalid' ? '' : 'MQTT_BROKER_URL'}
          error={state === 'invalid' ? t('env-vars:modalKeyRequired') : undefined}
          disabled={state === 'disabled'}
          readOnly={state === 'read-only'}
        />
      ),
    },
    {
      name: 'Select',
      render: (state: (typeof states)[number]) =>
        state === 'read-only' ? (
          na('n/a: a native select has no read-only state; use disabled.')
        ) : (
          <Select
            id={`sg-select-${state}`}
            label={t('env-vars:modalTypeLabel')}
            error={state === 'invalid' ? t('env-vars:modalValueMustBeBoolean') : undefined}
            disabled={state === 'disabled'}
          >
            {types}
          </Select>
        ),
    },
    {
      name: 'Textarea',
      render: (state: (typeof states)[number]) => (
        <Textarea
          id={`sg-textarea-${state}`}
          label={t('env-vars:modalDescriptionLabel')}
          rows={2}
          defaultValue={state === 'invalid' ? '' : t('configuration:topology.subtitle')}
          error={state === 'invalid' ? t('env-vars:modalValueRequired') : undefined}
          disabled={state === 'disabled'}
          readOnly={state === 'read-only'}
        />
      ),
    },
    {
      name: 'Toggle',
      render: (state: (typeof states)[number]) =>
        state === 'rest' ? (
          <div className="flex items-center gap-3">
            <Toggle checked={toggle} onCheckedChange={setToggle} aria-label={t('env-vars:modalBooleanValueLabel')} />
            <Toggle checked={false} onCheckedChange={noop} aria-label={t('env-vars:modalBooleanValueLabel')} />
            <span className="text-xs text-ds-text-secondary">on · off</span>
          </div>
        ) : state === 'disabled' ? (
          <Toggle checked disabled onCheckedChange={noop} aria-label={t('env-vars:modalBooleanValueLabel')} />
        ) : (
          na('n/a: a toggle is always valid and is disabled, not read-only.')
        ),
    },
  ];
  return (
    <Section
      title="Fields"
      intro="Raised-sheet ground, 1px control-border edge, 4px corners, 14px text. Focus turns the edge conductor blue; invalid draws a critical-red edge and names the problem under the field. Read-only keeps the edge on the sunken ground."
    >
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {columns.map((col) => (
          <div key={col.name} className="space-y-4 rounded-box border border-ds-border-default bg-ds-bg-surface p-4">
            <h3 className="text-sm font-semibold text-ds-text-primary">{col.name}</h3>
            {states.map((state) => (
              <div key={state} className="space-y-1">
                <p>
                  <Tag>{state}</Tag>
                </p>
                {col.render(state)}
              </div>
            ))}
          </div>
        ))}
      </div>
    </Section>
  );
}

export function DialogsSection() {
  const { t } = useT();
  const [dialog, setDialog] = useState(false);
  const [confirm, setConfirm] = useState(false);
  return (
    <Section title="Dialogs" intro="Modals are the last resort: use one only when the task needs protected focus. Portaled above the page, focus trapped, Escape closes, focus returns to the opener.">
      <Specimen name="Dialog" caption="Title, optional description, DialogBody and a DialogFooter on a hairline rule with the affirmative action last.">
        <Button type="button" variant="secondary" onClick={() => setDialog(true)}>
          Open dialog
        </Button>
        <Dialog open={dialog} onClose={() => setDialog(false)} title={t('env-vars:modalNewTitle')} description={t('env-vars:editorHelpText')}>
          <DialogBody className="space-y-4">
            <Input id="sg-dialog-key" label={t('env-vars:modalKeyLabel')} placeholder="MQTT_BROKER_URL" />
            <Select id="sg-dialog-type" label={t('env-vars:modalTypeLabel')}>
              <option>{t('env-vars:typeString')}</option>
            </Select>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="secondary" onClick={() => setDialog(false)}>
              {t('common:cancel')}
            </Button>
            <Button type="button" variant="solid" onClick={() => setDialog(false)}>
              {t('common:save')}
            </Button>
          </DialogFooter>
        </Dialog>
      </Specimen>
      <Specimen
        name="ConfirmationDialog"
        caption="Gates a destructive or risky action: states the consequence, can require the identifier to be typed, and confirms on the danger button. Focus starts on the safest target."
      >
        <Button type="button" variant="danger" onClick={() => setConfirm(true)}>
          Open confirmation
        </Button>
        <ConfirmationDialog
          isOpen={confirm}
          variant="danger"
          title={t('env-vars:deleteDialogTitle')}
          description={t('env-vars:deleteDialogDescription', { key: 'MQTT_BROKER_URL' })}
          confirmText="MQTT_BROKER_URL"
          onConfirm={() => setConfirm(false)}
          onCancel={() => setConfirm(false)}
        />
      </Specimen>
    </Section>
  );
}

export function TabsSection() {
  const { t } = useT();
  const [tab, setTab] = useState<'configured' | 'dotenv'>('configured');
  return (
    <Section title="Tabs" intro="A WAI-ARIA tablist with roving tabindex on a hairline baseline. Selected: conductor-blue text over a 2px underline. Arrow keys, Home and End move and select.">
      <Specimen name="Tabs and TabPanel" caption="Switch between views of the same object. Use a tablist, not buttons styled as tabs.">
        <Tabs
          label={t('env-vars:tabsLabel')}
          idBase="sg-tabs"
          items={[
            { id: 'configured', label: t('env-vars:tabConfigured'), icon: ListTree },
            { id: 'dotenv', label: t('env-vars:tabDotenv'), icon: FileText },
          ]}
          selected={tab}
          onSelect={setTab}
        />
        <TabPanel idBase="sg-tabs" selected={tab} className="pt-3 text-sm text-ds-text-secondary">
          {tab === 'configured' ? t('env-vars:noEnvVarsConfigured') : t('env-vars:dotenvEditorHint')}
        </TabPanel>
      </Specimen>
    </Section>
  );
}

const ROWS = [
  { key: 'MQTT_BROKER_URL', value: 'mqtt://broker.local:1883', type: 'typeString' },
  { key: 'POLL_INTERVAL_MS', value: '5000', type: 'typeNumber' },
  { key: 'API_TOKEN', value: '••••••••', type: 'typeSecret' },
] as const;

export function TablesSection() {
  const { t } = useT();
  const table = (stacked: boolean) => (
    <Table caption={t('env-vars:pageTitle')} stacked={stacked}>
      <thead>
        <tr>
          <th scope="col">{t('env-vars:colKey')}</th>
          <th scope="col">{t('env-vars:colValue')}</th>
          <th scope="col">{t('env-vars:colType')}</th>
        </tr>
      </thead>
      <tbody>
        {ROWS.map((row) => (
          <tr key={row.key}>
            <td data-label={t('env-vars:colKey')} className="font-mono text-xs font-semibold">
              {row.key}
            </td>
            <td data-label={t('env-vars:colValue')} className="font-mono text-xs">
              {row.value}
            </td>
            <td data-label={t('env-vars:colType')}>{t(`env-vars:${row.type}`)}</td>
          </tr>
        ))}
      </tbody>
    </Table>
  );
  return (
    <Section title="Tables" intro="Hairline row rules, 12px headers on the sunken ground, 14px cells, tabular numerals, dense rows.">
      <Specimen name="Table" caption="Dense data the operator scans by column. Scrolls sideways inside its frame when narrow." className="p-0">
        {table(false)}
      </Specimen>
      <Specimen
        name="Table, stacked"
        caption="Below 640px each row becomes a labelled card, so no column scrolls off a phone screen. Use it when every column matters on mobile (Environment, Users)."
        className="p-0"
      >
        {table(true)}
      </Specimen>
    </Section>
  );
}

export function StatusSection() {
  const { t } = useT();
  const chips = [
    { variant: 'success', label: t('common:appChrome.compatModeEditable') },
    { variant: 'info', label: t('configuration:circuit.state.active') },
    { variant: 'warning', label: t('common:appChrome.compatModeReadOnly') },
    { variant: 'danger', label: t('configuration:fieldStatus.validation.invalid') },
    { variant: 'neutral', label: t('common:appChrome.compatModeUnknown') },
  ] as const;
  return (
    <Section title="Status and feedback" intro="Every state in words first; colour and shape only confirm it.">
      <Specimen name="StatusChip" caption="The state of one item at a glance, 2px corners, label on a tint in the *-text colour. Green verified, brown pending, amber held, red failed, neutral unknown.">
        <div className="space-y-3">
          {(['sm', 'md'] as const).map((size) => (
            <div key={size} className="flex flex-wrap items-center gap-2">
              <span className="w-8"><Tag>{size}</Tag></span>
              {chips.map(({ variant, label }) => (
                <StatusChip key={variant} variant={variant} size={size}>
                  {label}
                </StatusChip>
              ))}
            </div>
          ))}
        </div>
      </Specimen>
      <div className="grid gap-6 lg:grid-cols-2">
        <Specimen name="EmptyState" caption="A query succeeded with zero rows: say what is missing and offer the one next action.">
          <EmptyState
            icon={<Inbox className="h-6 w-6" />}
            title={t('env-vars:noEnvVarsConfigured')}
            actionLabel={t('env-vars:add')}
            onAction={noop}
          />
        </Specimen>
        <Specimen name="ErrorState" caption="A query failed: name the problem; offer retry only when retrying is safe.">
          <ErrorState title={t('common:errorOccurred')} description={t('common:tryAgainLater')} onRetry={noop} />
        </Specimen>
      </div>
      <Specimen name="Skeleton" caption="Loading that keeps the layout: blocks on the sunken ground, one status region. The pulse stops under reduced motion.">
        <div className="grid items-start gap-4 sm:grid-cols-3">
          <Skeleton variant="text" rows={3} />
          <Skeleton variant="rect" />
          <Skeleton variant="circle" />
        </div>
      </Specimen>
      <Specimen name="StateContainer" caption="One branch for loading, error, empty and content, in that priority, with the default slots shown here.">
        <div className="grid gap-4 lg:grid-cols-4">
          {(['loading', 'error', 'empty', 'content'] as const).map((state) => (
            <div key={state} className="space-y-1">
              <Tag>{state}</Tag>
              <StateContainer isLoading={state === 'loading'} isError={state === 'error'} isEmpty={state === 'empty'}>
                <p className="py-10 text-center text-sm text-ds-text-primary">{t('configuration:circuit.proof.ready')}</p>
              </StateContainer>
            </div>
          ))}
        </div>
      </Specimen>
    </Section>
  );
}

const base: CircuitInput = { pendingCount: 0, invalidFields: [], reviewed: false, readOnly: false, run: { status: 'idle' } };

const CIRCUITS: { name: string; input: CircuitInput; restart: number }[] = [
  { name: 'idle', input: base, restart: 0 },
  { name: 'pending', input: { ...base, pendingCount: 3 }, restart: 1 },
  { name: 'review', input: { ...base, pendingCount: 3, reviewed: true }, restart: 1 },
  { name: 'passed', input: { ...base, run: { status: 'succeeded', count: 3 } }, restart: 0 },
  { name: 'rollback restored', input: { ...base, run: { status: 'failed', count: 3, code: 'RUNTIME_READINESS_FAILED' } }, restart: 0 },
  { name: 'recovery failed', input: { ...base, run: { status: 'failed', count: 3, code: 'RUNTIME_RECOVERY_FAILED' } }, restart: 0 },
];

export function CircuitSection() {
  return (
    <Section
      title="Change circuit"
      intro="The signature: Edit → Diff → Validate → Backup → Apply → Ready as six normally-open contacts on one conductor. A contact closes when its gate passes. Each state below comes from deriveCircuit with fixture input, not the API. Below 1024px it draws as a horizontal strip."
    >
      <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
        {CIRCUITS.map(({ name, input, restart }) => {
          const pending = input.run.status === 'idle' ? input.pendingCount : 0;
          return (
            <div key={name} className="min-w-0 space-y-2">
              <Tag>{name}</Tag>
              <div className="rounded-box border border-ds-border-default bg-ds-bg-surface p-4">
                <ChangeCircuit
                  circuit={deriveCircuit(input)}
                  pendingCount={pending}
                  restartCount={pending > 0 ? restart : 0}
                  reviewDisabled={pending === 0}
                  onReview={noop}
                  onDiscard={pending > 0 ? noop : undefined}
                  className="static border-b-0"
                />
              </div>
            </div>
          );
        })}
      </div>
    </Section>
  );
}
