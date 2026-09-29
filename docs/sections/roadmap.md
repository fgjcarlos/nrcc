<!--
  Roadmap — single source of truth mirrors docs/control-plane.md.
  Inlined into docs/index.html as the "Roadmap status" table.
  Whenever this table is regenerated, regenerate docs/control-plane.md
  with the same data — see scripts/check-pages-freshness.mjs for the
  CI guarantee.
-->

The 9 sub-clusters of the [control-plane roadmap](https://github.com/fgjcarlos/nrcc/issues/765)
track NRCC's progress towards a trustworthy Node-RED 5 control plane.
Status is mirrored from
[`docs/control-plane.md`](https://github.com/fgjcarlos/nrcc/blob/main/docs/control-plane.md).

| Cluster | Theme | Status |
|---------|-------|--------|
| [#756](https://github.com/fgjcarlos/nrcc/issues/756) | NR 5.x compatibility contract | 🟢 GREEN |
| [#757](https://github.com/fgjcarlos/nrcc/issues/757) | Settings.js source preservation | 🟢 GREEN |
| [#758](https://github.com/fgjcarlos/nrcc/issues/758) | Transactional settings apply | 🟢 GREEN |
| [#759](https://github.com/fgjcarlos/nrcc/issues/759) | Reliable access administration | 🟡 AMBER |
| [#760](https://github.com/fgjcarlos/nrcc/issues/760) | Authentication surfaces | 🟡 AMBER |
| [#761](https://github.com/fgjcarlos/nrcc/issues/761) | Dashboard access surfaces | 🟢 GREEN |
| [#762](https://github.com/fgjcarlos/nrcc/issues/762) | TLS, credentialSecret, requireHttps | 🟢 GREEN |
| [#763](https://github.com/fgjcarlos/nrcc/issues/763) | Navigation focused on configuration | 🟢 GREEN |
| [#764](https://github.com/fgjcarlos/nrcc/issues/764) | Advanced settings escape hatches | 🟡 AMBER |

**Overall:** 6 🟢 GREEN, 3 🟡 AMBER, 0 🔴 RED. The three AMBER items are
policy / evidence gaps, not implementation gaps — see
[`docs/control-plane.md`](https://github.com/fgjcarlos/nrcc/blob/main/docs/control-plane.md)
for per-cluster remediation.
