# Fix frontend build warnings

## Goal

Remove the two known frontend production build warnings so the build output is clean:
`@vueuse/core` pure annotation warnings from Rolldown and the oversized chunk warning.

## User Value

Build output should make real regressions visible. Known benign warnings should not keep appearing on every production build.

## Confirmed Facts

- `npm --prefix frontend run build` currently succeeds.
- The build still prints Rolldown `INVALID_ANNOTATION` warnings for `node_modules/@vueuse/core/dist/index.js`.
- The build prints Vite's chunk size warning because the previous production output put the dashboard JavaScript into one large `static/app/assets/app.js` file.
- This task is limited to the Vue/Vite frontend build path and generated static assets.

## Requirements

- Suppress or eliminate the Rolldown pure annotation warning without changing application behavior.
- Remove the oversized chunk warning by splitting the production JavaScript bundle.
- Keep `/static/app/assets/app.js` as the HTML entry script while allowing additional preload chunks.
- Keep the production build command unchanged: `npm --prefix frontend run build`.
- Do not introduce new runtime dependencies or broad frontend refactors.

## Acceptance Criteria

- [x] `npm --prefix frontend run build` exits successfully.
- [x] Build output no longer contains `INVALID_ANNOTATION`.
- [x] Build output no longer contains `Some chunks are larger than 500 kB`.
- [x] Generated JavaScript is split into multiple chunks while `app.js` remains the entry script.
- [x] `git diff --check` passes.

## Out Of Scope

- Full performance optimization of the JavaScript bundle.
- Dependency upgrades unless the existing config cannot solve the warnings.
