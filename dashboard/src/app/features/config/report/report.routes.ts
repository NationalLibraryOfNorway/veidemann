import {Routes} from '@angular/router';
import {
  CrawlExecutionComponent,
  CrawlLogComponent,
  JobExecutionComponent,
  PageLogComponent,
  ReportNavigationListComponent
} from '../../report/containers';
import {ReportComponent} from '../../report/report.component';
import {GuardService} from '../../../core/auth/guard.service';
import {OptionsResolver} from '../services';
import {CrawlExecutionDetailComponent} from '../../report/components';
import {JobExecutionDetailComponent} from '../../report/containers/job-execution-detail/job-execution-detail.component';
import {PageLogDetailComponent} from '../../report/containers/page-log-detail/page-log-detail.component';
import {CrawlLogDetailComponent} from '../../report/containers/crawl-log-detail/crawl-log-detail.component';


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
