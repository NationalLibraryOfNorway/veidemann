import { Routes } from '@angular/router';

import { GuardService } from '../../core';
import { CrawlExecutionDetailComponent } from './containers/crawl-execution-detail/crawl-execution-detail.component';
import { CrawlExecutionComponent } from './containers/crawl-execution/crawl-execution.component';
import { CrawlLogDetailComponent } from './containers/crawl-log-detail/crawl-log-detail.component';
import { CrawlLogComponent } from './containers/crawl-log/crawl-log.component';
import { JobExecutionDetailComponent } from './containers/job-execution-detail/job-execution-detail.component';
import { JobExecutionComponent } from './containers/job-execution/job-execution.component';
import { PageLogDetailComponent } from './containers/page-log-detail/page-log-detail.component';
import { PageLogComponent } from './containers/page-log/pagelog.component';
import { ReportNavigationListComponent } from './containers/report-navigation-list/report-navigation-list.component';
import { OptionsResolver } from './services/options.resolver.service';
import { ReportComponent } from './report.component';

export const routes: Routes = [
  {
    path: '',
    component: ReportNavigationListComponent
  },
  {
    path: '',
    component: ReportComponent,
    children: [
      {
        path: 'crawlexecution',
        component: CrawlExecutionComponent,
        canActivate: [GuardService],
        resolve: {
          options: OptionsResolver
        },
      },
      {
        path: 'crawlexecution/:id',
        component: CrawlExecutionDetailComponent,
      },
      {
        path: 'jobexecution',
        canActivate: [GuardService],
        component: JobExecutionComponent,
        resolve: {
          options: OptionsResolver
        },
      },
      {
        path: 'jobexecution/:id',
        component: JobExecutionDetailComponent,
      },
      {
        path: 'pagelog',
        canActivate: [GuardService],
        component: PageLogComponent,
      },
      {
        path: 'pagelog/:id',
        component: PageLogDetailComponent,
      },
      {
        path: 'crawllog',
        canActivate: [GuardService],
        component: CrawlLogComponent,
      },
      {
        path: 'crawllog/:id',
        component: CrawlLogDetailComponent
      }
    ]
  },
];
