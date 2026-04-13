import { ApplicationConfig, ErrorHandler, provideBrowserGlobalErrorListeners, APP_INITIALIZER } from '@angular/core';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { providePrimeNG } from 'primeng/config';
import Aura from '@primeuix/themes/aura';

import { MessageService } from 'primeng/api';
import { authInterceptor } from './core/auth.interceptor';
import { retryInterceptor } from './core/retry.interceptor';
import { GlobalErrorHandler } from './core/global-error-handler';
import { PriceAlertService } from './core/alerts/price-alert.service';
import { PortfolioAlertService } from './core/alerts/portfolio-alert.service';
import { UserPreferencesService } from './core/user-preferences.service';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    provideHttpClient(
      withInterceptors([retryInterceptor, authInterceptor]),
    ),
    MessageService,
    { provide: ErrorHandler, useClass: GlobalErrorHandler },
    {
      provide: APP_INITIALIZER,
      useFactory: (
        _priceAlerts: PriceAlertService,
        _portfolioAlerts: PortfolioAlertService,
        prefs: UserPreferencesService,
      ) => () => prefs.load().toPromise().catch(() => {}),
      deps: [PriceAlertService, PortfolioAlertService, UserPreferencesService],
      multi: true,
    },
    providePrimeNG({
      theme: {
        preset: Aura,
        options: {
          darkModeSelector: '.dark',
        },
      },
    }),
  ],
};
