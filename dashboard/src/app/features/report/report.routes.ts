import { Routes } from '@angular/router';
import { GuardService } from '../../core';
import { OptionsResolver } from './services';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    loadComponent: () =>
      import('./containers/report-navigation-list/report-navigation-list.component').then(
        m => m.ReportNavigationListComponent
      ),
  },
  {
    path: '',
    loadComponent: () =>
      import('./report.component').then(m => m.ReportComponent),
    children: [
      {
        path: 'crawlexecution',
        canActivate: [GuardService],
        resolve: { options: OptionsResolver },
        loadComponent: () =>
          import('./containers/crawl-execution/crawl-execution.component').then(
            m => m.CrawlExecutionComponent
          ),
      },
      {
        path: 'crawlexecution/:id',
        loadComponent: () =>
          import('./containers/crawl-execution-detail/crawl-execution-detail.component').then(
            m => m.CrawlExecutionDetailComponent
          ),
      },
      {
        path: 'jobexecution',
        canActivate: [GuardService],
        resolve: { options: OptionsResolver },
        loadComponent: () =>
          import('./containers/job-execution/job-execution.component').then(
            m => m.JobExecutionComponent
          ),
      },
      {
        path: 'jobexecution/:id',
        loadComponent: () =>
          import('./containers/job-execution-detail/job-execution-detail.component').then(
            m => m.JobExecutionDetailComponent
          ),
      },
      {
        path: 'pagelog',
        canActivate: [GuardService],
        loadComponent: () =>
          import('./containers/page-log/pagelog.component').then(
            m => m.PageLogComponent
          ),
      },
      {
        path: 'pagelog/:id',
        loadComponent: () =>
          import('./containers/page-log-detail/page-log-detail.component').then(
            m => m.PageLogDetailComponent
          ),
      },
      {
        path: 'crawllog',
        canActivate: [GuardService],
        loadComponent: () =>
          import('./containers/crawl-log/crawl-log.component').then(
            m => m.CrawlLogComponent
          ),
      },
      {
        path: 'crawllog/:id',
        loadComponent: () =>
          import('./containers/crawl-log-detail/crawl-log-detail.component').then(
            m => m.CrawlLogDetailComponent
          ),
      },
    ],
  },
];
