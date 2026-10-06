---
name: mobile-playwright-dnd
description: Exercise React HTML drag-and-drop handlers in the repository's mobile Playwright suite.
when_to_use: When an E2E scenario needs desktop drag-and-drop coverage while Playwright is configured with a touch/mobile device.
---

The project-wide Playwright configuration uses an iPhone device profile. A physical
`locator.dragTo()` gesture is touch-oriented there and may not invoke a component's
HTML `onDragStart`/`onDrop` handlers.

For a board implemented with native HTML drag events, dispatch the lifecycle directly:

```ts
await source.dispatchEvent("dragstart");
await target.dispatchEvent("drop");
```

Keep a separate keyboard-accessible control path in the same scenario. This proves
both interaction alternatives without changing the production component solely for
the test environment.
