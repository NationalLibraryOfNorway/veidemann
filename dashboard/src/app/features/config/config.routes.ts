import {OptionsResolver} from './services';
import {Routes} from '@angular/router';
import {GuardService} from '../../core';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    loadComponent: () =>
      import('./containers/config-nav-list/config-nav-list.component').then(
        (m) => m.ConfigNavListComponent
      ),
  },
  {
    path: ':kind',
    loadComponent: () =>
      import('./config').then((m) => m.ConfigComponent),
    resolve: { options: OptionsResolver },
    canActivate: [GuardService],
    children: [
      {
        path: '',
        loadComponent: () =>
          import('./containers/configurations/configurations.component').then(
            (m) => m.ConfigurationsComponent
          ),
      },
      {
        path: ':id',
        loadComponent: () =>
          import('./containers/configuration/configuration.component').then(
            (m) => m.ConfigurationComponent
          ),
      },
    ],
  },
];

