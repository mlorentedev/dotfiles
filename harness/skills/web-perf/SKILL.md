---
generated: true
generated_from: 00_meta/skills/web-perf/SKILL.md
generated_sha: 0cca326c02fce669
id: web-perf-skill
type: skill
status: active
created: "2026-11-07"
owner: manu
name: web-perf
description: Audit, diagnose, or optimize website loading and interaction performance, Core Web Vitals (LCP, CLS, INP, TTFB), and bundle size. Auto-triggers proactively on web frontend changes before completion.
source: https://github.com/cloudflare/skills
license: Apache-2.0
keywords: [web performance, core web vitals, lcp, cls, inp, lighthouse, render blocking, bundle size, optimize web, layout shift]
paths: ['**/*.tsx', '**/*.jsx', '**/*.vue', '**/*.svelte', '**/*.astro', '**/pages/**', '**/components/**', '**/routes/**', '**/styles/**']
---

# Web Performance Audit & Optimization

Audit, diagnose, and optimize web applications for loading speed, interactivity, and visual stability against Core Web Vitals targets.

> **Origin:** adapted from [cloudflare/skills](https://github.com/cloudflare/skills) (`skills/web-perf`, Apache-2.0). Adapted for the cross-agent skill pipeline with autonomous frontend activation rules and pattern-web-performance thresholds.

## Core Web Vitals Thresholds (75th Percentile)

| Metric | Target (Good) | Needs Work | Poor | Priority Culprits |
|---|---|---|---|---|
| **LCP** (Largest Contentful Paint) | **< 2.5s** | 2.5s – 4.0s | > 4.0s | Large unoptimized hero assets, render-blocking CSS/JS, slow TTFB |
| **CLS** (Cumulative Layout Shift) | **< 0.1** | 0.1 – 0.25 | > 0.25 | Images/embeds without explicit dimensions, dynamic DOM injections without reserved skeleton |
| **INP** (Interaction to Next Paint) | **< 200ms** | 200ms – 500ms | > 500ms | Long JS tasks (>50ms) on main thread, synchronous event handlers, layout thrashing |
| **TTFB** (Time to First Byte) | **< 800ms** | 800ms – 1.8s | > 1.8s | Cold starts, missing edge cache headers, synchronous backend blocking |

---

## Agent-Side Activation Rule (Autonomous Trigger)

> **Proactive Mode.** This skill triggers reactively on `/web-perf [url]`, but also runs **proactively** without requiring human initiation.

### Proactive Checks (Run Before Completing Frontend Changes)

When an agent finishes writing or modifying frontend code (`.tsx`, `.jsx`, `.vue`, `.svelte`, `.astro`, `.css`, or UI components) in web applications (e.g. `fae-onboarding-webapp`):

1. **Explicit Media Dimensions**:
   - Check every newly added or modified `<img>`, `<video>`, `<iframe>`, or canvas.
   - If missing `width`/`height` or CSS `aspect-ratio` → **Flag CLS risk immediately**.
2. **Render-Blocking Scripts**:
   - Check `<script>` tags in HTML/templates or entrypoint bundles.
   - If importing non-critical libraries synchronously in critical render paths → **Flag LCP/TTFB risk**. Recommend dynamic `import()` or `defer`/`async`.
3. **Dynamic Content Skeletons**:
   - Check dynamic DOM insertions or async data fetches (e.g., banners, dropdowns, cards).
   - If rendered into containers without minimum height or placeholder skeleton → **Flag CLS risk**.
4. **Asset & Dependency Bloat**:
   - Check package imports (e.g., `import { X } from 'large-lib'`).
   - If full library is pulled when tree-shakeable submodule exists → **Flag bundle weight finding**.

When any risk is detected, the agent proactively outputs a Web Performance block and proposes the immediate code remedy before marking the task done.

---

## Audit Workflow

### Phase 1: Tooling Verification & Trace Capture
1. Check if browser / performance tracking tools (such as Chrome DevTools MCP) are available:
   - If DevTools MCP is available:
     - Load page: `navigate_page(url: "<url>")`
     - Start trace: `performance_start_trace(autoStop: true, reload: true)`
   - If browser tools are unavailable:
     - Fall back to static source, asset, and bundle analysis (inspect Vite/Webpack/Next.js configs, inspect network waterfall if accessible). Explicitly state which trace metrics were estimated via static analysis.

### Phase 2: Core Web Vitals Analysis
1. **LCP Breakdown**:
   - Identify the LCP element (image, text block, video poster).
   - Measure TTFB vs Resource Load Delay vs Resource Load Duration vs Render Delay.
   - Target: If TTFB is >40% of LCP, optimize backend/caching. If Load Duration is >40%, compress asset/convert to WebP/AVIF.
2. **CLS Culprits**:
   - Pinpoint layout shift culprits:
     - Images or video elements missing `width`/`height`.
     - Injected banners or notifications pushing content down.
     - Webfont swap (FOUT) causing text reflow.

### Phase 3: Network & Asset Inspection
1. **Render-Blocking Resources**:
   - Identify CSS and JS files requested before First Contentful Paint.
   - Verify if any can be inlined (critical CSS) or deferred.
2. **Compression & Caching**:
   - Verify static assets serve with `Cache-Control: public, max-age=31536000, immutable`.
   - Verify text assets serve with Brotli (`br`) or Gzip compression.

### Phase 4: Codebase & Bundler Optimization
1. Inspect bundler configuration (Vite, Next.js, Rollup):
   - Are vendor chunks separated?
   - Are heavy charting, editor, or utility libraries code-split using React `lazy()`, Vue `defineAsyncComponent()`, or dynamic imports?
2. Quantify the estimated savings: Ignore 0ms micro-optimizations; focus on changes that save >100ms or reduce bundle by >20KB.

---

## Output Format

When generating an audit report, output the following structured markdown:

```markdown
### Web Performance Audit

**Target**: `<url or path>`
**Evaluator**: `<agent or tool>`

#### Core Web Vitals Assessment
| Metric | Observed / Estimated | Status (Good / Warning / Poor) | Primary Culprit |
|---|---|---|---|
| LCP | | | |
| CLS | | | |
| INP | | | |
| TTFB | | | |

#### Prioritized Findings
1. **[High] <Element/Asset>**: <Issue description> (Estimated savings: ~Xms / Y KB)
   - **Remedy**: <Exact code or config snippet>
2. **[Medium] <Element/Asset>**: <Issue description>
   - **Remedy**: <Exact code or config snippet>

#### Verification Command
`<command or test proving the fix, e.g. bundle analyzer or lighthouse run>`
```
