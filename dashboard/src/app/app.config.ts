import { HttpClient, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { ApplicationConfig, ErrorHandler, importProvidersFrom, inject, Injectable, LOCALE_ID, provideAppInitializer, provideBrowserGlobalErrorListeners, provideZonelessChangeDetection, signal } from '@angular/core';
import { NoPreloading, provideRouter, withPreloading } from '@angular/router';

import { DateFnsAdapter, MAT_DATE_FNS_FORMATS } from '@angular/material-date-fns-adapter';
import { DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE } from '@angular/material/core';
import { MAT_FORM_FIELD_DEFAULT_OPTIONS } from '@angular/material/form-field';

import { createMongoAbility, PureAbility } from '@casl/ability';
import { AuthConfig, JwksValidationHandler, OAuthModule, OAuthService, ValidationHandler } from 'angular-oauth2-oidc';
import { KeyboardShortcutsModule } from 'ng-keyboard-shortcuts';

import { routes } from './app.routes';
import { ApplicationErrorHandler, AuthService, ControllerApiService, LocaleService } from './core';
import { firstValueFrom } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AppConfig {
  authConfig: AuthConfig;
  grpcWebUrl: string;


  load(json?: Partial<AppConfig>) {
    if (json) {
      Object.assign(this, json);
    }
  }
}

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),
    provideRouter(routes, withPreloading(NoPreloading)),
    provideHttpClient(withInterceptorsFromDi()),
    importProvidersFrom(
      OAuthModule.forRoot(),
      KeyboardShortcutsModule.forRoot(),
    ),
    OAuthService,
    { provide: ValidationHandler, useClass: JwksValidationHandler },
    { provide: PureAbility, useValue: createMongoAbility() },
    provideAppInitializer(initializeApp),
    {
      provide: LOCALE_ID,
      useFactory: (localeService: LocaleService) => localeService.getLocale(),
      deps: [LocaleService]
    },
    {
      provide: MAT_DATE_LOCALE,
      useFactory: (localeService: LocaleService) => localeService.getLocale(),
      deps: [LocaleService]
    },
    { provide: DateAdapter, useClass: DateFnsAdapter, deps: [MAT_DATE_LOCALE] },
    { provide: MAT_FORM_FIELD_DEFAULT_OPTIONS, useValue: { floatLabel: 'auto' } },
    { provide: MAT_DATE_FORMATS, useValue: MAT_DATE_FNS_FORMATS },
    { provide: ErrorHandler, useClass: ApplicationErrorHandler },
  ]
}

export const initError = signal<Error | null>(null);

export async function initializeApp() {
      const controllerApiService = inject(ControllerApiService);
      const oAuthService = inject(OAuthService);
      const authService = inject(AuthService);
      const appConfig = inject(AppConfig);
      const http = inject(HttpClient);

      try {
        const dynamicConfig = await firstValueFrom(http.get<AppConfig>('public/config.json'));
        appConfig.load(dynamicConfig);
        const issuer = await controllerApiService.getOpenIdConnectIssuer();

        if (issuer) {
          appConfig.authConfig.issuer = issuer;
          oAuthService.configure(new AuthConfig(appConfig.authConfig));
          await oAuthService.loadDiscoveryDocumentAndTryLogin();
          if (!oAuthService.hasValidIdToken()) {
            oAuthService.logOut(true);
          }
        }

        authService.roles = await controllerApiService.getRolesForActiveUser();
        authService.updateAbility();
      } catch (error) {
        initError.set(error);
      }
}
