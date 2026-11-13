# Review changes panel: confirm, feedback and single submit

The Apply button in the configuration "Review changes" drawer gives no feedback, can be
clicked repeatedly, and applies without a confirmation step.

## Specs

- S1: "deberia de tener un doble check" — Apply in the review drawer asks for confirmation before saving. The credentialSecret rotation path, which already has its own confirmation dialog, must not ask twice.
- S2: "tneer feedback al user de que se esta realziando la accion" — while the apply request runs, the user sees it in progress (drawer Apply shows "Aplicando…", the confirmation shows its processing state, the global Save button shows saving).
- S3: "desabilitarse al pulsar una vez para evitar puslar 2 veces o mas" — after the first confirmed click, Apply/Confirm are disabled until the request settles; a double click sends exactly one request.

## Tasks

- T1 (S1-S3) worker — use the save hook's own pending state (ConfigurationView passes actions.saveConfigMutation.isPending, but useConfigurationSave runs a separate mutation instance), add a ConfirmationDialog step, update unit + e2e tests. Commit: "fix(configuration): confirm and track the review panel apply".

## Log

- L1: "Bien pero el boton de sheet apply, no tieen accion, esd decir deberia de tener un doble check, y tneer feedback al user de que se esta realziando la accion, ademas de desabilitarse al pulsar una vez para evitar puslar 2 veces o mas"
- L2: Exploration: root cause ConfigurationView.tsx:509 passes actions.saveConfigMutation.isPending while useConfigurationSave.ts:45 creates its own useConfigurationActions() mutation (:71 mutateAsync, :77 isPending unused). Reusable dialog: shared/components/ConfirmationDialog.tsx (isPending support). e2e/stack.spec.ts ~855-858 clicks Apply and expects the success toast.
