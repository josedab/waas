# WaaS Dashboard

Web dashboard for the Webhook-as-a-Service platform. Built with React, TypeScript, Vite, and Tailwind CSS.

## Prerequisites

- Node.js 18+
- pnpm (`npm install -g pnpm`)

## Setup

```bash
# Install dependencies (must use pnpm - npm/yarn will fail due to lockfile)
pnpm install

# Copy environment config
cp .env.example .env.local
```

## Development

```bash
pnpm dev        # Start dev server (default: http://localhost:5173)
pnpm build      # Production build (runs tsc + vite build)
pnpm preview    # Preview production build locally
pnpm lint       # Run ESLint
pnpm test       # Run tests with Vitest
```

## Environment Variables

Create `.env.local` with:

```env
VITE_API_URL=http://localhost:8080/api/v1
```

## Project Structure

```
src/
├── components/     # Reusable UI components
├── pages/          # Route-level page components
├── services/       # API client and service layer
├── stores/         # Zustand state management
├── types/          # TypeScript type definitions
└── utils/          # Shared utilities
```
