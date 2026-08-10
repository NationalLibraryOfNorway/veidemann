import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideNativeDateAdapter} from '@angular/material/core';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, ParamMap, Router} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {BehaviorSubject, EMPTY, of} from 'rxjs';

import {ControllerApiService, SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlExecutionService} from '../../services';
import {CrawlExecutionComponent} from './crawl-execution.component';

describe('CrawlExecutionComponent', () => {
  let fixture: ComponentFixture<CrawlExecutionComponent>;
  let component: CrawlExecutionComponent;
  let queryParams: BehaviorSubject<ParamMap>;

  const navigate = vi.fn<Router['navigate']>().mockResolvedValue(true);

  beforeEach(async () => {
    queryParams = new BehaviorSubject(convertToParamMap({
      sort: 'startTime:desc',
      p: '2',
      s: '50',
    }));
    navigate.mockClear();

    await TestBed.configureTestingModule({
      imports: [CrawlExecutionComponent],
      providers: [
        ...provideCoreTesting,
        provideNativeDateAdapter(),
        {
          provide: ActivatedRoute,
          useValue: {
            queryParamMap: queryParams,
            snapshot: {data: {options: {crawlJobs: []}}},
          },
        },
        {provide: Router, useValue: {navigate, events: EMPTY}},
        {
          provide: CrawlExecutionService,
          useValue: {
            search: () => EMPTY,
            loading$: of(false),
          },
        },
        {provide: MatDialog, useValue: {open: vi.fn()}},
        {provide: ControllerApiService, useValue: {}},
        {provide: SnackBarService, useValue: {}},
        {provide: AbilityServiceSignal, useValue: {can: () => false}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlExecutionComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('preserves sorting while applying filters and clearing legacy paging', () => {
    expect(component.sortActive()).toBe('startTime');
    expect(component.sortDirection()).toBe('desc');

    component.onQueryChange({...component.query(), jobId: 'job-1'});

    expect(navigate).toHaveBeenCalledTimes(1);
    const [, options] = navigate.mock.calls[0];
    expect(options.queryParamsHandling).toBe('merge');
    expect(options.queryParams).toEqual(expect.objectContaining({
      p: null,
      s: null,
      job_id: 'job-1',
    }));
    expect(options.queryParams['sort']).toBeUndefined();
  });
});
