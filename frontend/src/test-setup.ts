// Vitest setup for Angular testing
import { getTestBed } from '@angular/core/testing';
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting,
} from '@angular/platform-browser-dynamic/testing';
import { expect, vi } from 'vitest';

// Initialize TestBed for Angular tests
try {
  getTestBed().initTestEnvironment(BrowserDynamicTestingModule, platformBrowserDynamicTesting(), {
    teardown: { destroyAfterEach: true },
  });
} catch (e) {
  // Ignore if already initialized
}

const createCompatSpy = () => {
  const spy = vi.fn();
  (spy as any).and = {
    returnValue: (value: unknown) => {
      spy.mockReturnValue(value);
      return spy;
    },
    callFake: (impl: (...args: unknown[]) => unknown) => {
      spy.mockImplementation(impl);
      return spy;
    },
  };
  return spy;
};

(globalThis as any).jasmine = {
  createSpy: (_name?: string) => createCompatSpy(),
  createSpyObj: (
    _baseName: string,
    methodNames: string[],
    properties?: Record<string, unknown>,
  ) => {
    const out: Record<string, unknown> = {};
    for (const methodName of methodNames) {
      out[methodName] = createCompatSpy();
    }
    if (properties) {
      Object.assign(out, properties);
    }
    return out;
  },
  objectContaining: <T>(sample: Partial<T>) => expect.objectContaining(sample),
  stringContaining: (sample: string) => expect.stringContaining(sample),
};

(globalThis as any).spyOn = vi.spyOn;
(globalThis as any).fail = (message?: string) => {
  throw new Error(message ?? 'Test failed');
};

if (!window.matchMedia) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
}

