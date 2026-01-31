# @waas/portal

Embeddable webhook management portal for SaaS companies. Drop-in React component with endpoint CRUD, delivery logs, theming via CSS variables, and white-label support.

## Quick Start

```tsx
import { WaaSPortal } from '@waas/portal';

function App() {
  return (
    <WaaSPortal
      config={{
        apiUrl: 'https://your-waas-api.com',
        token: 'embed_your_token_here',
      }}
    />
  );
}
```

## Theming

Customize the portal via CSS variables or the `theme` prop:

```tsx
<WaaSPortal
  config={{ apiUrl: '...', token: '...' }}
  theme={{
    primaryColor: '#0066ff',
    backgroundColor: '#1a1a2e',
    textColor: '#e0e0e0',
    fontFamily: '"Inter", sans-serif',
    borderRadius: '12px',
  }}
/>
```

Available CSS variables: `--waas-primary`, `--waas-bg`, `--waas-surface`, `--waas-text`, `--waas-muted`, `--waas-success`, `--waas-error`, `--waas-warning`, `--waas-radius`, `--waas-font`.

## Feature Flags

```tsx
<WaaSPortal
  config={{
    apiUrl: '...',
    token: '...',
    features: {
      endpoints: true,
      deliveries: true,
      testSender: false,
      analytics: false,
    },
    maxEndpoints: 10,
  }}
/>
```

## Callbacks

```tsx
<WaaSPortal
  config={{ apiUrl: '...', token: '...' }}
  onEndpointCreated={(ep) => console.log('Created:', ep)}
  onEndpointDeleted={(id) => console.log('Deleted:', id)}
  onError={(err) => console.error('Portal error:', err)}
/>
```
