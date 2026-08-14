import {ErrorHandler} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {of} from 'rxjs';

import {provideCoreTesting} from '../../../../core/core.testing.module';
import {PageLog} from '../../../../shared/models';
import {PageLogService} from '../../services';
import {PageLogComponent} from './pagelog.component';

describe('PageLogComponent', () => {
  let fixture: ComponentFixture<PageLogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PageLogComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {provide: PageLogService, useValue: {
          loading$: of(false),
          search: () => of(new PageLog({warcId: 'warc-1', uri: 'https://example.org'})),
        }},
        {provide: ErrorHandler, useValue: {handleError: vi.fn()}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(PageLogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders the list without a filter toolbar or execution actions menu', () => {
    expect(fixture.nativeElement.querySelector('app-pagelog-list').textContent)
      .toContain('https://example.org');
    expect(fixture.nativeElement.querySelector('.report-filter-toolbar')).toBeNull();
    const executionQuery = fixture.nativeElement.querySelector('app-page-log-query') as HTMLElement;
    expect(executionQuery).not.toBeNull();
    expect((executionQuery.querySelector('mat-form-field') as HTMLElement).hidden).toBe(true);
    expect(fixture.nativeElement.querySelector('app-log-list-shortcuts')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();
  });

  it('keeps execution-ID query changes wired to the route', () => {
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);

    fixture.componentInstance.onQueryChange({...fixture.componentInstance.query(), executionId: 'execution-2'});

    expect(navigate).toHaveBeenCalledWith([], expect.objectContaining({
      queryParamsHandling: 'merge',
      queryParams: expect.objectContaining({
        p: null,
        s: null,
        execution_id: 'execution-2',
        job_execution_id: null,
        uri: null,
      }),
    }));
  });
});
