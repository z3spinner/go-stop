import '@testing-library/jest-dom/vitest';
import { afterAll } from 'vitest';

// bits-ui dialogs restore the body's scroll lock on a timer after they unmount.
// When a test file ends with a dialog open, that timer can fire after jsdom has
// been torn down ("document is not defined" in body-scroll-lock), which Vitest
// reports as an unhandled error and fails the run. Give pending timers a moment
// to run while the DOM still exists.
afterAll(() => new Promise((resolve) => setTimeout(resolve, 300)));
