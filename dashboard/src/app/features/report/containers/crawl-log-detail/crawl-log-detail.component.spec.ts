import {Clipboard} from '@angular/cdk/clipboard';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {ActivatedRoute, convertToParamMap, provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';
import {of} from 'rxjs';

import {SnackBarService} from '../../../../core';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {CrawlLog} from '../../../../shared/models';
import {CrawlLogService} from '../../services';
import {CrawlLogDetailComponent} from './crawl-log-detail.component';

describe('CrawlLogDetailComponent', () => {
  let fixture: ComponentFixture<CrawlLogDetailComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CrawlLogDetailComponent],
      providers: [
        ...provideCoreTesting,
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: of(convertToParamMap({id: 'warc-1'})),
            queryParamMap: of(convertToParamMap({})),
          },
        },
        {provide: CrawlLogService, useValue: {get: () => of(new CrawlLog({warcId: 'warc-1'}))}},
        {provide: AbilityServiceSignal, useValue: {can: () => true}},
        {provide: Clipboard, useValue: {copy: () => true}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn(), openError: vi.fn()}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlLogDetailComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
  });

  it('renders the full-width detail table without a header or overflow actions', () => {
    expect(fixture.nativeElement.querySelector('app-crawl-log-status')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('table.report-table')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-header')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-detail-overflow')).toBeNull();
    expect(fixture.nativeElement.querySelector('app-crawl-log-shortcuts')).toBeNull();
  });
});
