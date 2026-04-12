// Vitest setup for Angular testing
import { getTestBed } from '@angular/core/testing';

// Initialize TestBed for Angular tests
try {
  getTestBed().initTestEnvironment(
    // Use the default platform-browser environment without @angular/platform-browser-dynamic
    // This is sufficient for Vitest with jsdom
    {} as any,
    {} as any,
    { teardownTestingModule: { destroyAfterEach: true } } as any,
  );
} catch (e) {
  // Ignore if already initialized
}

