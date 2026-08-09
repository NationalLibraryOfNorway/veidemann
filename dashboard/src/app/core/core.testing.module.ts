import {ErrorHandler, Provider, provideZonelessChangeDetection} from '@angular/core';
import { MATERIAL_ANIMATIONS } from '@angular/material/core';
import { of } from 'rxjs';
import {AuthService, ConfigApiService, GuardService, SnackBarService} from '.';
import { AbilityServiceSignal } from '@casl/angular';
import { AppConfig } from '../app.config';

export function provideMaterialAnimationsDisabled(): Provider {
  return { provide: MATERIAL_ANIMATIONS, useValue: { animationsDisabled: true } };
}

export const provideCoreTesting = [
  provideZonelessChangeDetection(),
  provideMaterialAnimationsDisabled(),
  {
    provide: AppConfig,
    useValue: {
      grpcWebUrl: 'http://localhost:8080',
    },
  },
  {
    provide: ConfigApiService,
    useValue: {
      list: () => of(null),
      getLabelKeys: () => of([]),
    },
  },
  { provide: GuardService, useValue: {} },
  {
    provide: AbilityServiceSignal,
    useValue: {
      can: () => false,
    },
  },
  {
    provide: AuthService,
    useValue: {
      isAdmin: () => true,
      isCurator: () => true,
      canCreate: () => true,
      canUpdate: () => true,
      canDelete: () => true,
    },
  },
  {provide: ErrorHandler, useValue: {handleError: () => undefined}},
  { provide: SnackBarService, useValue: {} },
];
