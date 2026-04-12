import { TestBed } from '@angular/core/testing';
import { ThemeService } from './theme.service';

describe('ThemeService', () => {
  let service: ThemeService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    localStorage.clear();
    document.documentElement.classList.remove('dark');
    service = TestBed.inject(ThemeService);
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove('dark');
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should default to light mode', () => {
    expect(service.isDark()).toBe(false);
  });

  it('should toggle theme', () => {
    expect(service.isDark()).toBe(false);
    service.toggle();
    expect(service.isDark()).toBe(true);
    service.toggle();
    expect(service.isDark()).toBe(false);
  });

  it('should set theme explicitly', () => {
    service.setDark(true);
    expect(service.isDark()).toBe(true);
    service.setDark(false);
    expect(service.isDark()).toBe(false);
  });

  it('should add dark class to document.documentElement when dark', () => {
    service.setDark(true);
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('should remove dark class from document.documentElement when light', () => {
    document.documentElement.classList.add('dark');
    service.setDark(false);
    expect(document.documentElement.classList.contains('dark')).toBe(false);
  });

  it('should persist theme to localStorage', () => {
    service.setDark(true);
    expect(localStorage.getItem('app-theme')).toBe('dark');
    service.setDark(false);
    expect(localStorage.getItem('app-theme')).toBe('light');
  });

  it('should load theme from localStorage on init', () => {
    localStorage.setItem('app-theme', 'dark');
    const newService = TestBed.inject(ThemeService);
    expect(newService.isDark()).toBe(true);
  });
});
