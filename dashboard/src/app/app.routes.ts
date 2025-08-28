import { Routes } from '@angular/router';
import { GuardService } from './core';
import { AppComponent } from './features/app/app.component';

export const routes: Routes = [
  {
    path: '',
    component: AppComponent,
    children: [
      {
        path: '',
        pathMatch: 'full',
        loadChildren: () => import('./features/app/app.routes').then(m => m.routes),
      },
      {
        path: 'config',
        loadChildren: () => import('./features/config/config.routes').then(m => m.routes),
        canActivate: [GuardService],
      },
      {
        path: 'report',
        loadChildren: () => import('./features/report/report.routes').then(m => m.routes),
        canActivate: [GuardService],
      },
      {
        path: 'logconfig',
        loadChildren: () => import('./features/log/log.routes').then(m => m.routes),
        canActivate: [GuardService],
      },
    ],
  },
];
