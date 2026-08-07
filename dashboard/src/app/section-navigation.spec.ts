import {Component} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {provideRouter, Routes} from '@angular/router';
import {RouterTestingHarness} from '@angular/router/testing';

import {provideMaterialAnimationsDisabled} from './core/core.testing.module';
import {ConfigComponent} from './features/config/config';
import {OptionsService} from './features/config/services';
import {ReportComponent} from './features/report/report.component';

@Component({
  template: '<span data-testid="section-index">Section index</span>',
  standalone: true,
})
class SectionIndexComponent {}

@Component({
  template: '<span data-testid="section-page">Section page</span>',
  standalone: true,
})
class SectionPageComponent {}

const routes: Routes = [
  {
    path: 'config',
    children: [
      {path: '', pathMatch: 'full', component: SectionIndexComponent},
      {
        path: ':kind',
        component: ConfigComponent,
        data: {options: {}},
        children: [
          {path: '', pathMatch: 'full', component: SectionPageComponent},
        ],
      },
    ],
  },
  {
    path: 'report',
    children: [
      {path: '', pathMatch: 'full', component: SectionIndexComponent},
      {
        path: '',
        component: ReportComponent,
        children: [
          {path: 'jobexecution', component: SectionPageComponent},
          {path: 'crawlexecution', component: SectionPageComponent},
        ],
      },
    ],
  },
];

describe('section navigation', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      providers: [
        provideMaterialAnimationsDisabled(),
        provideRouter(routes),
        {provide: OptionsService, useValue: {next: vi.fn()}},
      ],
    }).compileComponents();
  });

  it('loads the first child after entering and switching sections', async () => {
    const harness = await RouterTestingHarness.create('/config');

    await harness.navigateByUrl('/config/entity');
    expectPage(harness);

    await harness.navigateByUrl('/report');
    await harness.navigateByUrl('/report/jobexecution');
    expectPage(harness);

    await harness.navigateByUrl('/report/crawlexecution');
    expectPage(harness);

    await harness.navigateByUrl('/config');
    await harness.navigateByUrl('/config/seed');
    expectPage(harness);
  });
});

function expectPage(harness: RouterTestingHarness): void {
  const element = harness.routeNativeElement as HTMLElement;

  expect(element.querySelector('[data-testid="section-page"]')).not.toBeNull();
}
