import {ConfigurationComponent, ConfigurationsComponent} from './containers';
import {OptionsResolver} from './services';
import {ConfigComponent} from './containers';
import {ConfigNavListComponent} from './containers/config-nav-list/config-nav-list.component';
import { Routes } from '@angular/router';
import { GuardService } from '../../core';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    component: ConfigNavListComponent,
  },
  {
    path: ':kind',
    component: ConfigComponent,
    resolve: {
      options: OptionsResolver
    },
    canActivate: [GuardService],
    children: [
      {
        path: '',
        component: ConfigurationsComponent,
      },
      {
        path: ':id',
        component: ConfigurationComponent,
      },
    ],
  },

];

