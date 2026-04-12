import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { AppShellComponent } from './app-shell.component';
import { AuthService } from '../../../core/auth.service';
import { ThemeService } from '../../../core/theme.service';

describe('AppShellComponent', () => {
  let component: AppShellComponent;
  let fixture: ComponentFixture<AppShellComponent>;
  let authService: jasmine.SpyObj<AuthService>;
  let themeService: jasmine.SpyObj<ThemeService>;
  let router: jasmine.SpyObj<Router>;

  beforeEach(async () => {
    const authServiceMock = jasmine.createSpyObj('AuthService', ['logout'], {
      user: jasmine.createSpy().and.returnValue(null),
    });
    const themeServiceMock = jasmine.createSpyObj('ThemeService', ['toggle'], {
      isDark: jasmine.createSpy().and.returnValue(false),
    });
    const routerMock = jasmine.createSpyObj('Router', ['navigate'], {
      url: '/',
      events: jasmine.createSpyObj('events', ['subscribe']),
    });

    await TestBed.configureTestingModule({
      imports: [AppShellComponent],
      providers: [
        { provide: AuthService, useValue: authServiceMock },
        { provide: ThemeService, useValue: themeServiceMock },
        { provide: Router, useValue: routerMock },
      ],
    }).compileComponents();

    authService = TestBed.inject(AuthService) as jasmine.SpyObj<AuthService>;
    themeService = TestBed.inject(ThemeService) as jasmine.SpyObj<ThemeService>;
    router = TestBed.inject(Router) as jasmine.SpyObj<Router>;

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
    expect(authService.logout).toHaveBeenCalled();
  });

  it('should build menu items without admin section for non-admin users', () => {
    fixture.detectChanges();
    const items = component.menuItems();
    expect(items.length).toBeGreaterThan(0);
    expect(items.some(item => item.label === 'Dashboard')).toBe(true);
    expect(items.some(item => item.label === 'Portfolios')).toBe(true);
  });
});
