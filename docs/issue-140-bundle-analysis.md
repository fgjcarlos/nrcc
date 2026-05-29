# Issue #140 bundle analysis

Measured with `cd frontend && pnpm run build` on this branch worktree.

## Before route-level lazy loading (`origin/main`)

```text
dist/index.html                     1.29 kB │ gzip:   0.65 kB
dist/assets/index-CmPNSSJl.css    141.62 kB │ gzip:  21.71 kB
dist/assets/index-VnweaJjs.js   1,028.64 kB │ gzip: 297.17 kB
```

The initial JavaScript bundle eagerly included all feature views and triggered Vite's >500 kB chunk warning.

## After route-level lazy loading

```text
dist/index.html                              1.29 kB │ gzip:   0.65 kB
dist/assets/index-DyJaJ8HX.css             141.99 kB │ gzip:  21.78 kB
dist/assets/index-CzNMgec-.js              415.20 kB │ gzip: 131.74 kB
dist/assets/DashboardView-DfjBHNpX.js      353.57 kB │ gzip: 104.03 kB
dist/assets/schemas-ewJY3RJj.js             87.07 kB │ gzip:  25.85 kB
dist/assets/ConfigurationView-DRIURZk0.js   26.98 kB │ gzip:   6.69 kB
dist/assets/BackupsView-DfbSGK8D.js         25.27 kB │ gzip:   6.44 kB
dist/assets/FlowDetailView-CeDcT1hh.js      22.22 kB │ gzip:   5.76 kB
dist/assets/EnvVarsView-DdXhNp48.js         15.40 kB │ gzip:   4.45 kB
dist/assets/UsersView-PuRL7NGD.js           11.08 kB │ gzip:   2.84 kB
dist/assets/LandingView-D2UHU4Yx.js         10.73 kB │ gzip:   3.51 kB
dist/assets/LibrariesView-DziQORJm.js       10.20 kB │ gzip:   3.02 kB
dist/assets/BootstrapView-CTkTmmL_.js       10.00 kB │ gzip:   2.05 kB
dist/assets/UpdatesView-BJlHygKO.js          9.98 kB │ gzip:   2.55 kB
dist/assets/FlowVersionsView-B1DA479U.js     5.81 kB │ gzip:   1.71 kB
dist/assets/DockerView-Za0mKcWQ.js           5.08 kB │ gzip:   1.49 kB
dist/assets/ProfileView-c5gjiXy4.js          4.69 kB │ gzip:   1.44 kB
dist/assets/LogsView-DDhJXloa.js             4.52 kB │ gzip:   1.85 kB
dist/assets/LoginView--an19B_x.js            3.84 kB │ gzip:   1.35 kB
dist/assets/SetupView-gUJy_4FE.js            3.65 kB │ gzip:   1.31 kB
```

## Result

- Initial JS decreased from 1,028.64 kB to 415.20 kB minified.
- Initial JS gzip decreased from 297.17 kB to 131.74 kB.
- Reduction: 613.44 kB minified (about 59.6%) and 165.43 kB gzip (about 55.7%).
- Major feature views are now emitted as separate route chunks and load behind route-level Suspense fallbacks.
- The previous Vite large initial chunk warning no longer appears for the entry bundle.
