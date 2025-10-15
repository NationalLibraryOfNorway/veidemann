import { Routes } from '@angular/router';
import { LoglevelComponent } from './log';
import { LogResolver } from './services';


export const routes: Routes = [
  {
    path: '',
    component: LoglevelComponent,
    resolve: {
      levels: LogResolver
    },
  },
];
