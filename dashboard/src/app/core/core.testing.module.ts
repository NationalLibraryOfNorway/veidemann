import { provideZonelessChangeDetection } from '@angular/core';
import { of } from 'rxjs';
import { AuthService, ConfigApiService, ErrorService, GuardService, SnackBarService } from '.';
import { AbilityService } from '@casl/angular';
import { AppConfig } from '../app.config';

export const provideCoreTesting = [
  provideZonelessChangeDetection(),
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
    },
  },
  { provide: GuardService, useValue: {} },
  {
    provide: AbilityService,
    useValue: {
      ability$: of(null),
    },
  },
  {
    provide: AuthService,
    useValue: {
      isAdmin: () => true,
      isCurator: () => true,
      canUpdate: () => true,
      canDelete: () => true,
    },
  },
  { provide: ErrorService, useValue: {} },
  { provide: SnackBarService, useValue: {} },
];


// import {ModuleWithProviders, NgModule, provideZonelessChangeDetection} from '@angular/core';
//
// import {AuthService, ConfigApiService, ErrorService, GuardService, SnackBarService} from '.';
// import {of} from 'rxjs';
// import {AbilityService} from "@casl/angular";
// import { AppConfig } from '../app.config';
//
// @NgModule()
// export class CoreTestingModule {
//   static forRoot(): ModuleWithProviders<CoreTestingModule> {
//     return {
//       ngModule: CoreTestingModule,
//       providers: [
//         provideZonelessChangeDetection(),
//         {
//           provide: AppConfig,
//           useValue: {
//             grpcWebUrl: 'http://localhost:8080',
//           }
//         },
//         {
//           provide: ConfigApiService,
//           useValue: {
//             list: () => of(null)
//           }
//         },
//         {
//           provide: GuardService,
//           useValue: {}
//         },
//         {
//           provide: AbilityService,
//           useValue: {
//             ability$: of(null)
//           }
//         },
//         {
//           provide: AuthService,
//           useValue: {
//             isAdmin: () => true,
//             isCurator: () => true,
//             canUpdate: () => true,
//             canDelete: () => true,
//           }
//         },
//         {
//           provide: ErrorService,
//           useValue: {}
//         },
//         {
//           provide: SnackBarService,
//           useValue: {}
//         }
//       ]
//     };
//   }
// }
