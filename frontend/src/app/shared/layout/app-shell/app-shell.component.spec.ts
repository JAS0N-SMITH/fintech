import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { Router } from '@angular/router';
import { AppShellComponent } from './app-shell.component';
import { AuthService } from '../../../core/auth.service';
import { ThemeService } from '../../../core/theme.service';

describe('AppShellComponent', () => {
  let component: AppShellComponent;
  let fixture: ComponentFixture<AppShellComponent>;
  let authService: any;
  let themeService: any;
  let router: any;

  beforeEach(async () => {
    const authServiceMock = {
      signOut: vi.fn(),
      user: signal(null).asReadonly(),
    };
    const themeServiceMock = {
      toggle: vi.fn(),
      isDark: vi.fn().mockReturnValue(false),
    };
    const routerMock = {
      navigate: vi.fn(),
      url: '/',
      events: { subscribe: vi.fn() },
    };

    await TestBed.configureTestingModule({
      imports: [AppShellComponent],
      providers: [
        { provide: AuthService, useValue: authServiceMock },
        { provide: ThemeService, useValue: themeServiceMock },
        { provide: Router, useValue: routerMock },
      ],
    })
      .overrideComponent(AppShellComponent, {
        set: { template: '<div></div>' },
      })
      .compileComponents();

    authService = TestBed.inject(AuthService) as any;
    themeService = TestBed.inject(ThemeService) as any;
    router = TestBed.inject(Router) as any;

    fixture = TestBed.createComponent(AppShellComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should toggle sidebar', () => {
    expect(component.isSidebarCollapsed()).toBe(false);
    component.toggleSidebar();
    expect(component.isSidebarCollapsed()).toBe(true);
    component.toggleSidebar();
    expect(component.isSidebarCollapsed()).toBe(false);
  });

  it('should toggle theme', () => {
    component.toggleTheme();
    expect(themeService.toggle).toHaveBeenCalled();
  });

  it('should logout', () => {
    component.logout();
    expect(authService['signOut']).toHaveBeenCalled();
  });

  it('should build menu items without admin section for non-admin users', () => {
    fixture.detectChanges();
    const items = component.menuItems();
    expect(items.length).toBeGreaterThan(0);
    expect(items.some(item => item.label === 'Dashboard')).toBe(true);
    expect(items.some(item => item.label === 'Portfolios')).toBe(true);
  });
});
