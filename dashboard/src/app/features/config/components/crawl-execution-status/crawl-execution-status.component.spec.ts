import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionStatusComponent} from './crawl-execution-status.component';
import {CrawlExecutionStatus} from '../../../../shared/models';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';
import {of} from 'rxjs';

describe('CrawlExecutionStatusComponent', () => {
  let component: CrawlExecutionStatusComponent;
  let fixture: ComponentFixture<CrawlExecutionStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusComponent],
      providers: [
        ...provideCoreTesting,
        { provide: ActivatedRoute, useValue: { snapshot: {}, params: of({}), queryParams: of({}) } }
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionStatusComponent);
    component = fixture.componentInstance;
    component.crawlExecutionStatus = new CrawlExecutionStatus();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

});
