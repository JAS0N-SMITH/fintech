declare namespace jasmine {
  type SpyObj<T> = T & Record<string, any>;

  function createSpy(name?: string): any;
  function createSpyObj<T>(
    baseName: string,
    methodNames: string[],
    properties?: Record<string, unknown>,
  ): SpyObj<T>;
  function objectContaining<T>(sample: Partial<T>): any;
  function stringContaining(sample: string): any;
}

declare function spyOn<T extends object, K extends keyof T>(obj: T, method: K): any;
declare function fail(message?: string): never;
