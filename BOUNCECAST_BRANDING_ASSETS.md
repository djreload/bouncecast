# BounceCast Branding Asset Checklist

This pass updates safe visible text first. It does not invent final BounceCast image assets.

## Current Asset Locations

| Asset | Current path | Notes |
| --- | --- | --- |
| Favicon | `static/favicon.png` | Runtime default favicon served by the backend. Needs final BounceCast replacement. |
| Public logo | `static/img/logo.png` | Runtime default public logo. Needs final BounceCast replacement. |
| Web source logo | `web/assets/images/logo.svg` | Source logo used by frontend components. Needs final BounceCast SVG. |
| Admin/common logo component | `web/components/common/OwncastLogo/OwncastLogo.tsx` | Component name is internal for now; visible rendered asset should be replaced after a BounceCast logo exists. |
| Public manifest | `web/public/manifest.json` | App display name updated to BounceCast. Icon still points at `/favicon.ico`. |
| Built manifest | `static/web/manifest.json` | App display name updated to BounceCast. Regenerate from frontend build when assets are final. |
| Storybook brand assets | `web/.storybook/story-assets/project/` | Owncast design/reference assets. Leave until a full design-system pass. |
| Open Graph image | Not clearly present as a dedicated source asset | Add a final BounceCast OG image when the visual identity exists. |
| Splash/social image references | README previously referenced upstream Owncast splash image | README now avoids using upstream splash as BounceCast branding. |

## Needed Final Assets

| Needed asset | Suggested dimensions/formats | Target paths |
| --- | --- | --- |
| Favicon | 32x32 PNG and/or ICO | `static/favicon.png`, optionally `web/public/favicon.ico` if added later |
| App icon | 192x192 and 512x512 PNG | Add under `web/public/` and update manifests |
| Public logo | SVG preferred, PNG fallback | `web/assets/images/logo.svg`, `static/img/logo.png` |
| Admin logo | SVG preferred | Render through `web/components/common/OwncastLogo/OwncastLogo.tsx` or a future renamed wrapper |
| Open Graph image | 1200x630 PNG | Add a dedicated asset and wire metadata once metadata ownership is clear |
| Documentation screenshots | PNG/WebP at source resolution | Replace only when BounceCast UI screenshots exist |
| Docker/release image references | Repository/image metadata, not a visual asset | Keep binary/container internals stable until release process is reviewed |

## TODO

- Create final BounceCast logo artwork.
- Replace `web/assets/images/logo.svg` with the final source SVG.
- Replace `static/img/logo.png` and `static/favicon.png`.
- Add app icons and update `web/public/manifest.json`.
- Run `cd web && npm run build` to regenerate `static/web`.
- Review Storybook project assets after the runtime app is stable.
- Review Open Graph/social metadata after the public branding assets exist.
