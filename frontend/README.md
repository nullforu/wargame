# Frontend of WARGAME Wargame Platform

- See [nullforu/wargame](https://github.com/nullforu/wargame) for the backend of the platform.
- See [nullforu/wargame-docs](https://github.com/nullforu/wargame-docs) for the documentation of the platform.

## Environment Variables

- `VITE_API_BASE` (default: `http://localhost:8080`)
- `VITE_S3_CHALLENGE_UPLOAD_PRESIGN_METHOD` (default: `POST`, supported: `POST`, `PUT`)
- `VITE_S3_MEDIA_CDN_BASE_URL` (example: `https://wargame-cdn.swua.kr`)

Profile image UX notes:

- Upload flow starts from `Upload Image` button (opens file picker).
- After file selection, a square crop modal opens immediately.
- Pressing `Apply` in modal uploads immediately (no extra upload step).
- Upload uses media presigned `POST` only and enforces max `100KB`.
- Stored DB value is object key only (e.g. `profiles/<uuid>.png`), and frontend renders with `VITE_S3_MEDIA_CDN_BASE_URL + key`.

```shell
VITE_API_BASE=http://localhost:8080 \
VITE_S3_CHALLENGE_UPLOAD_PRESIGN_METHOD=PUT \
VITE_S3_MEDIA_CDN_BASE_URL=https://wargame-cdn.swua.kr \
npm run dev

VITE_API_BASE=https://internal.swua.kr/wargame \
VITE_S3_CHALLENGE_UPLOAD_PRESIGN_METHOD=PUT \
VITE_S3_MEDIA_CDN_BASE_URL=https://wargame-cdn.swua.kr \
npm run build

wrangler pages deploy dist \
  --project-name null4u-wargame \
  --branch=main
```

<!-- # React + TypeScript + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) (or [oxc](https://oxc.rs) when used in [rolldown-vite](https://vite.dev/guide/rolldown)) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

## React Compiler

The React Compiler is currently not compatible with SWC. See [this issue](https://github.com/vitejs/vite-plugin-react/issues/428) for tracking the progress.

## Expanding the ESLint configuration

If you are developing a production application, we recommend updating the configuration to enable type-aware lint rules:

```js
export default defineConfig([
    globalIgnores(['dist']),
    {
        files: ['**/*.{ts,tsx}'],
        extends: [
            // Other configs...

            // Remove tseslint.configs.recommended and replace with this
            tseslint.configs.recommendedTypeChecked,
            // Alternatively, use this for stricter rules
            tseslint.configs.strictTypeChecked,
            // Optionally, add this for stylistic rules
            tseslint.configs.stylisticTypeChecked,

            // Other configs...
        ],
        languageOptions: {
            parserOptions: {
                project: ['./tsconfig.node.json', './tsconfig.app.json'],
                tsconfigRootDir: import.meta.dirname,
            },
            // other options...
        },
    },
])
```

You can also install [eslint-plugin-react-x](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-x) and [eslint-plugin-react-dom](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-dom) for React-specific lint rules:

```js
// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default defineConfig([
    globalIgnores(['dist']),
    {
        files: ['**/*.{ts,tsx}'],
        extends: [
            // Other configs...
            // Enable lint rules for React
            reactX.configs['recommended-typescript'],
            // Enable lint rules for React DOM
            reactDom.configs.recommended,
        ],
        languageOptions: {
            parserOptions: {
                project: ['./tsconfig.node.json', './tsconfig.app.json'],
                tsconfigRootDir: import.meta.dirname,
            },
            // other options...
        },
    },
])
``` -->
