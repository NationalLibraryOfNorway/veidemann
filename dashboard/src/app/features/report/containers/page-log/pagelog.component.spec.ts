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
  let component: PageLogComponent;

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
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('keeps URI display while clearing unsupported URI and Job Execution filters', () => {
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);

    expect(fixture.nativeElement.querySelector('app-pagelog-list').textContent)
      .toContain('https://example.org');

    component.onExecutionFilterRemove();

    expect(navigate).toHaveBeenCalledWith([], expect.objectContaining({
      queryParamsHandling: 'merge',
      queryParams: expect.objectContaining({
        p: null,
        s: null,
        execution_id: null,
        job_execution_id: null,
        uri: null,
      }),
    }));
    expect(navigate.mock.calls[0][1]?.queryParams?.['sort']).toBeUndefined();
  });

  it('places filtered-log actions at the top right of the filter toolbar', () => {
    const header = fixture.nativeElement.querySelector('.report-filter-header') as HTMLElement;
    const controls = header.querySelector('app-page-log-query') as HTMLElement;
    const shortcuts = header.querySelector('app-log-list-shortcuts') as HTMLElement;

    expect(getComputedStyle(header).display).toBe('grid');
    expect(getComputedStyle(controls).gridColumnStart).toBe('1');
    expect(getComputedStyle(shortcuts).display).toBe('contents');
  });
});
