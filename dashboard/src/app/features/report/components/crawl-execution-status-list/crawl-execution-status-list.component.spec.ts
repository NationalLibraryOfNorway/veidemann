import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlExecutionStatusListComponent} from './crawl-execution-status-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlExecutionStatusListComponent', () => {
  let component: CrawlExecutionStatusListComponent;
  let fixture: ComponentFixture<CrawlExecutionStatusListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusListComponent],
      providers: [
        ...provideCoreTesting,
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlExecutionStatusListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
